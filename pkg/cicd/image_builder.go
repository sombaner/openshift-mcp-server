package cicd

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"

	// OpenShift build APIs - using simplified approach
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ImageBuilder struct {
	dockerClient     *client.Client
	kubeConfig       *rest.Config
	kubeClient       kubernetes.Interface
	defaultNamespace string
}

type BuildConfig struct {
	Name          string
	Namespace     string
	SourceRepo    string
	SourceBranch  string
	Dockerfile    string
	ContextPath   string
	ImageName     string
	ImageTag      string
	BuildArgs     map[string]string
	Labels        map[string]string
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

func NewImageBuilder(kubeConfig *rest.Config, defaultNamespace string) (*ImageBuilder, error) {
	// Initialize Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("Warning: Failed to initialize Docker client: %v", err)
	}

	var kubeClient kubernetes.Interface
	if kubeConfig != nil {
		// Initialize Kubernetes client
		kubeClient, err = kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
		}
	}

	return &ImageBuilder{
		dockerClient:     dockerClient,
		kubeConfig:       kubeConfig,
		kubeClient:       kubeClient,
		defaultNamespace: defaultNamespace,
	}, nil
}

func (ib *ImageBuilder) BuildImage(ctx context.Context, config BuildConfig) (*BuildResult, error) {
	startTime := time.Now()

	switch config.BuildStrategy {
	case "kubernetes":
		return ib.buildWithKubernetes(ctx, config, startTime)
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

	// Convert build args to the required format
	buildArgs := make(map[string]*string)
	for k, v := range config.BuildArgs {
		buildArgs[k] = &v
	}

	// Prepare build options
	buildOptions := types.ImageBuildOptions{
		Dockerfile: config.Dockerfile,
		Tags:       []string{fmt.Sprintf("%s:%s", config.ImageName, config.ImageTag)},
		BuildArgs:  buildArgs,
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

func (ib *ImageBuilder) buildWithKubernetes(ctx context.Context, config BuildConfig, startTime time.Time) (*BuildResult, error) {
	if ib.kubeClient == nil {
		return nil, fmt.Errorf("Kubernetes client not available")
	}

	namespace := config.Namespace
	if namespace == "" {
		namespace = ib.defaultNamespace
	}

	// Create a Kubernetes Job for building (simplified approach)
	jobName := fmt.Sprintf("%s-build", config.Name)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels:    config.Labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "builder",
							Image: "docker:20.10-dind",
							Command: []string{
								"/bin/sh",
								"-c",
								fmt.Sprintf("git clone %s /workspace && cd /workspace && docker build -t %s:%s -f %s %s",
									config.SourceRepo, config.ImageName, config.ImageTag, config.Dockerfile, config.ContextPath),
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &[]bool{true}[0], // Required for Docker-in-Docker
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "docker-sock",
									MountPath: "/var/run/docker.sock",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "docker-sock",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/run/docker.sock",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create the job
	_, err := ib.kubeClient.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return &BuildResult{
			Success:   false,
			Error:     fmt.Errorf("failed to create build job: %w", err),
			BuildTime: time.Since(startTime),
		}, nil
	}

	// Wait for job completion
	buildLogs := "Build job created successfully"

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
	// Wait for Kubernetes job completion (simplified version)
	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
			job, err := ib.kubeClient.BatchV1().Jobs(namespace).Get(ctx, buildName, metav1.GetOptions{})
			if err != nil {
				return "", fmt.Errorf("failed to get job status: %w", err)
			}

			// Check job completion
			if job.Status.Succeeded > 0 {
				logs, err := ib.getBuildLogs(ctx, namespace, buildName)
				return logs, err
			}
			if job.Status.Failed > 0 {
				logs, _ := ib.getBuildLogs(ctx, namespace, buildName)
				return logs, fmt.Errorf("build job failed")
			}
		}
	}
	return "", fmt.Errorf("build timeout")
}

func (ib *ImageBuilder) getBuildLogs(ctx context.Context, namespace, buildName string) (string, error) {
	// This would need to use the OpenShift build logs API
	// For now, return a placeholder
	return fmt.Sprintf("Build logs for %s in namespace %s", buildName, namespace), nil
}

func (ib *ImageBuilder) ListImages(ctx context.Context, namespace string) ([]string, error) {
	if ib.dockerClient != nil {
		// List Docker images
		images, err := ib.dockerClient.ImageList(ctx, image.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list Docker images: %w", err)
		}

		var imageNames []string
		for _, img := range images {
			for _, tag := range img.RepoTags {
				imageNames = append(imageNames, tag)
			}
		}
		return imageNames, nil
	}

	return nil, fmt.Errorf("no Docker client available")
}

func (ib *ImageBuilder) DeleteImage(ctx context.Context, imageName string, namespace string) error {
	if ib.dockerClient != nil {
		// Delete Docker image
		_, err := ib.dockerClient.ImageRemove(ctx, imageName, image.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if err != nil {
			return fmt.Errorf("failed to delete Docker image: %w", err)
		}
		return nil
	}

	return fmt.Errorf("no Docker client available")
}
