package cicd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	buildv1 "github.com/openshift/api/build/v1"
	buildclientv1 "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	imageclientv1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type ImageBuilder struct {
	dockerClient     *client.Client
	openshiftConfig  *rest.Config
	buildClient      buildclientv1.BuildV1Interface
	imageClient      imageclientv1.ImageV1Interface
	defaultNamespace string
}

type BuildConfig struct {
	Name         string
	Namespace    string
	SourceRepo   string
	SourceBranch string
	Dockerfile   string
	ContextPath  string
	ImageName    string
	ImageTag     string
	BuildArgs    map[string]string
	Labels       map[string]string
	BuildStrategy string // "docker" or "openshift"
}

type BuildResult struct {
	ImageName     string
	ImageTag      string
	FullImageName string
	BuildTime     time.Duration
	BuildLogs     string
	Success       bool
	Error         error
}

func NewImageBuilder(openshiftConfig *rest.Config, defaultNamespace string) (*ImageBuilder, error) {
	// Initialize Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("Warning: Failed to initialize Docker client: %v", err)
	}

	var buildClient buildclientv1.BuildV1Interface
	var imageClient imageclientv1.ImageV1Interface

	if openshiftConfig != nil {
		// Initialize OpenShift clients
		buildClientset, err := buildclientv1.NewForConfig(openshiftConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenShift build client: %w", err)
		}
		buildClient = buildClientset

		imageClientset, err := imageclientv1.NewForConfig(openshiftConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenShift image client: %w", err)
		}
		imageClient = imageClientset
	}

	return &ImageBuilder{
		dockerClient:     dockerClient,
		openshiftConfig:  openshiftConfig,
		buildClient:      buildClient,
		imageClient:      imageClient,
		defaultNamespace: defaultNamespace,
	}, nil
}

func (ib *ImageBuilder) BuildImage(ctx context.Context, config BuildConfig) (*BuildResult, error) {
	startTime := time.Now()

	switch config.BuildStrategy {
	case "openshift":
		return ib.buildWithOpenShift(ctx, config, startTime)
	case "docker":
		fallthrough
	default:
		return ib.buildWithDocker(ctx, config, startTime)
	}
}

func (ib *ImageBuilder) buildWithDocker(ctx context.Context, config BuildConfig, startTime time.Time) (*BuildResult, error) {
	if ib.dockerClient == nil {
		return nil, fmt.Errorf("Docker client not available")
	}

	// Create build context
	buildContext, err := ib.createBuildContext(config.ContextPath, config.Dockerfile)
	if err != nil {
		return &BuildResult{
			Success:   false,
			Error:     fmt.Errorf("failed to create build context: %w", err),
			BuildTime: time.Since(startTime),
		}, nil
	}
	defer buildContext.Close()

	// Prepare build options
	buildOptions := types.ImageBuildOptions{
		Dockerfile: config.Dockerfile,
		Tags:       []string{fmt.Sprintf("%s:%s", config.ImageName, config.ImageTag)},
		BuildArgs:  config.BuildArgs,
		Labels:     config.Labels,
		Remove:     true,
		Context:    buildContext,
	}

	// Start build
	response, err := ib.dockerClient.ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		return &BuildResult{
			Success:   false,
			Error:     fmt.Errorf("failed to start Docker build: %w", err),
			BuildTime: time.Since(startTime),
		}, nil
	}
	defer response.Body.Close()

	// Read build logs
	buildLogs, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("Warning: Failed to read build logs: %v", err)
	}

	return &BuildResult{
		ImageName:     config.ImageName,
		ImageTag:      config.ImageTag,
		FullImageName: fmt.Sprintf("%s:%s", config.ImageName, config.ImageTag),
		BuildTime:     time.Since(startTime),
		BuildLogs:     string(buildLogs),
		Success:       true,
		Error:         nil,
	}, nil
}

func (ib *ImageBuilder) buildWithOpenShift(ctx context.Context, config BuildConfig, startTime time.Time) (*BuildResult, error) {
	if ib.buildClient == nil {
		return nil, fmt.Errorf("OpenShift build client not available")
	}

	namespace := config.Namespace
	if namespace == "" {
		namespace = ib.defaultNamespace
	}

	// Create ImageStream if it doesn't exist
	imageStreamName := config.ImageName
	imageStream := &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imageStreamName,
			Namespace: namespace,
		},
		Spec: imagev1.ImageStreamSpec{
			LookupPolicy: imagev1.ImageLookupPolicy{
				Local: true,
			},
		},
	}

	_, err := ib.imageClient.ImageStreams(namespace).Create(ctx, imageStream, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return &BuildResult{
			Success:   false,
			Error:     fmt.Errorf("failed to create ImageStream: %w", err),
			BuildTime: time.Since(startTime),
		}, nil
	}

	// Create BuildConfig
	buildConfigName := fmt.Sprintf("%s-build", config.Name)
	buildConfig := &buildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildConfigName,
			Namespace: namespace,
			Labels:    config.Labels,
		},
		Spec: buildv1.BuildConfigSpec{
			Source: buildv1.BuildSource{
				Type: buildv1.BuildSourceGit,
				Git: &buildv1.GitBuildSource{
					URI: config.SourceRepo,
					Ref: config.SourceBranch,
				},
				ContextDir: config.ContextPath,
			},
			Strategy: buildv1.BuildStrategy{
				Type: buildv1.DockerBuildStrategyType,
				DockerStrategy: &buildv1.DockerBuildStrategy{
					DockerfilePath: config.Dockerfile,
					Env: func() []corev1.EnvVar {
						var envVars []corev1.EnvVar
						for k, v := range config.BuildArgs {
							envVars = append(envVars, corev1.EnvVar{
								Name:  k,
								Value: v,
							})
						}
						return envVars
					}(),
				},
			},
			Output: buildv1.BuildOutput{
				To: &corev1.ObjectReference{
					Kind: "ImageStreamTag",
					Name: fmt.Sprintf("%s:%s", imageStreamName, config.ImageTag),
				},
			},
		},
	}

	// Create or update BuildConfig
	_, err = ib.buildClient.BuildConfigs(namespace).Create(ctx, buildConfig, metav1.CreateOptions{})
	if err != nil && strings.Contains(err.Error(), "already exists") {
		_, err = ib.buildClient.BuildConfigs(namespace).Update(ctx, buildConfig, metav1.UpdateOptions{})
	}
	if err != nil {
		return &BuildResult{
			Success:   false,
			Error:     fmt.Errorf("failed to create/update BuildConfig: %w", err),
			BuildTime: time.Since(startTime),
		}, nil
	}

	// Start a new build
	buildRequest := &buildv1.BuildRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildConfigName,
		},
	}

	build, err := ib.buildClient.BuildConfigs(namespace).Instantiate(ctx, buildConfigName, buildRequest, metav1.CreateOptions{})
	if err != nil {
		return &BuildResult{
			Success:   false,
			Error:     fmt.Errorf("failed to start build: %w", err),
			BuildTime: time.Since(startTime),
		}, nil
	}

	// Wait for build completion and get logs
	buildLogs, err := ib.waitForBuildCompletion(ctx, namespace, build.Name)
	if err != nil {
		return &BuildResult{
			Success:   false,
			Error:     fmt.Errorf("build failed: %w", err),
			BuildTime: time.Since(startTime),
			BuildLogs: buildLogs,
		}, nil
	}

	return &BuildResult{
		ImageName:     config.ImageName,
		ImageTag:      config.ImageTag,
		FullImageName: fmt.Sprintf("%s:%s", config.ImageName, config.ImageTag),
		BuildTime:     time.Since(startTime),
		BuildLogs:     buildLogs,
		Success:       true,
		Error:         nil,
	}, nil
}

func (ib *ImageBuilder) createBuildContext(contextPath, dockerfile string) (io.ReadCloser, error) {
	// If contextPath is empty, use current directory
	if contextPath == "" {
		contextPath = "."
	}

	// Create tar archive of the build context
	return archive.TarWithOptions(contextPath, &archive.TarOptions{
		Compression:     archive.Uncompressed,
		ExcludePatterns: []string{".git", ".gitignore", "*.md"},
	})
}

func (ib *ImageBuilder) waitForBuildCompletion(ctx context.Context, namespace, buildName string) (string, error) {
	// Poll build status
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
			build, err := ib.buildClient.Builds(namespace).Get(ctx, buildName, metav1.GetOptions{})
			if err != nil {
				return "", fmt.Errorf("failed to get build status: %w", err)
			}

			switch build.Status.Phase {
			case buildv1.BuildPhaseComplete:
				// Get build logs
				logs, err := ib.getBuildLogs(ctx, namespace, buildName)
				return logs, err
			case buildv1.BuildPhaseFailed:
				logs, _ := ib.getBuildLogs(ctx, namespace, buildName)
				return logs, fmt.Errorf("build failed")
			case buildv1.BuildPhaseCancelled:
				logs, _ := ib.getBuildLogs(ctx, namespace, buildName)
				return logs, fmt.Errorf("build cancelled")
			case buildv1.BuildPhaseError:
				logs, _ := ib.getBuildLogs(ctx, namespace, buildName)
				return logs, fmt.Errorf("build error")
			}
		}
	}
}

func (ib *ImageBuilder) getBuildLogs(ctx context.Context, namespace, buildName string) (string, error) {
	// This would need to use the OpenShift build logs API
	// For now, return a placeholder
	return fmt.Sprintf("Build logs for %s in namespace %s", buildName, namespace), nil
}

func (ib *ImageBuilder) ListImages(ctx context.Context, namespace string) ([]string, error) {
	if namespace == "" {
		namespace = ib.defaultNamespace
	}

	if ib.imageClient != nil {
		// List OpenShift ImageStreams
		imageStreams, err := ib.imageClient.ImageStreams(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list ImageStreams: %w", err)
		}

		var images []string
		for _, is := range imageStreams.Items {
			for _, tag := range is.Status.Tags {
				images = append(images, fmt.Sprintf("%s:%s", is.Name, tag.Tag))
			}
		}
		return images, nil
	}

	if ib.dockerClient != nil {
		// List Docker images
		images, err := ib.dockerClient.ImageList(ctx, types.ImageListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list Docker images: %w", err)
		}

		var imageNames []string
		for _, image := range images {
			for _, tag := range image.RepoTags {
				imageNames = append(imageNames, tag)
			}
		}
		return imageNames, nil
	}

	return nil, fmt.Errorf("no image client available")
}

func (ib *ImageBuilder) DeleteImage(ctx context.Context, imageName string, namespace string) error {
	if namespace == "" {
		namespace = ib.defaultNamespace
	}

	if ib.imageClient != nil {
		// Delete OpenShift ImageStream
		parts := strings.Split(imageName, ":")
		imageStreamName := parts[0]
		
		err := ib.imageClient.ImageStreams(namespace).Delete(ctx, imageStreamName, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete ImageStream: %w", err)
		}
		return nil
	}

	if ib.dockerClient != nil {
		// Delete Docker image
		_, err := ib.dockerClient.ImageRemove(ctx, imageName, types.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if err != nil {
			return fmt.Errorf("failed to delete Docker image: %w", err)
		}
		return nil
	}

	return fmt.Errorf("no image client available")
}
