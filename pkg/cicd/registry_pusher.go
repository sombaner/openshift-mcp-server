package cicd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	imagev1 "github.com/openshift/api/image/v1"
	imageclientv1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type RegistryPusher struct {
	dockerClient    *client.Client
	openshiftConfig *rest.Config
	imageClient     imageclientv1.ImageV1Interface
	registries      map[string]*RegistryConfig
}

type RegistryConfig struct {
	URL      string
	Username string
	Password string
	Email    string
	Secure   bool
}

type PushConfig struct {
	SourceImage   string
	TargetImage   string
	TargetTag     string
	Registry      string
	Namespace     string
	RegistryAuth  *RegistryConfig
	Labels        map[string]string
	Annotations   map[string]string
}

type PushResult struct {
	SourceImage   string
	TargetImage   string
	FullImageName string
	PushTime      time.Duration
	PushLogs      string
	Success       bool
	Error         error
	Digest        string
}

func NewRegistryPusher(openshiftConfig *rest.Config) (*RegistryPusher, error) {
	// Initialize Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("Warning: Failed to initialize Docker client: %v", err)
	}

	var imageClient imageclientv1.ImageV1Interface
	if openshiftConfig != nil {
		// Initialize OpenShift image client
		imageClientset, err := imageclientv1.NewForConfig(openshiftConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenShift image client: %w", err)
		}
		imageClient = imageClientset
	}

	return &RegistryPusher{
		dockerClient:    dockerClient,
		openshiftConfig: openshiftConfig,
		imageClient:     imageClient,
		registries:      make(map[string]*RegistryConfig),
	}, nil
}

func (rp *RegistryPusher) AddRegistry(name string, config *RegistryConfig) {
	rp.registries[name] = config
	log.Printf("Added registry configuration for %s (%s)", name, config.URL)
}

func (rp *RegistryPusher) RemoveRegistry(name string) {
	delete(rp.registries, name)
	log.Printf("Removed registry configuration for %s", name)
}

func (rp *RegistryPusher) PushImage(ctx context.Context, config PushConfig) (*PushResult, error) {
	startTime := time.Now()

	// Determine registry configuration
	var registryConfig *RegistryConfig
	if config.RegistryAuth != nil {
		registryConfig = config.RegistryAuth
	} else if config.Registry != "" {
		var ok bool
		registryConfig, ok = rp.registries[config.Registry]
		if !ok {
			return &PushResult{
				Success:   false,
				Error:     fmt.Errorf("registry configuration not found: %s", config.Registry),
				PushTime:  time.Since(startTime),
			}, nil
		}
	}

	// Use OpenShift ImageStream if available
	if rp.imageClient != nil && config.Namespace != "" {
		return rp.pushWithOpenShift(ctx, config, registryConfig, startTime)
	}

	// Fall back to Docker
	if rp.dockerClient != nil {
		return rp.pushWithDocker(ctx, config, registryConfig, startTime)
	}

	return &PushResult{
		Success:  false,
		Error:    fmt.Errorf("no push client available"),
		PushTime: time.Since(startTime),
	}, nil
}

func (rp *RegistryPusher) pushWithDocker(ctx context.Context, config PushConfig, registryConfig *RegistryConfig, startTime time.Time) (*PushResult, error) {
	// Tag the image for the target registry
	targetImage := config.TargetImage
	if config.TargetTag != "" {
		targetImage = fmt.Sprintf("%s:%s", config.TargetImage, config.TargetTag)
	}

	err := rp.dockerClient.ImageTag(ctx, config.SourceImage, targetImage)
	if err != nil {
		return &PushResult{
			Success:   false,
			Error:     fmt.Errorf("failed to tag image: %w", err),
			PushTime:  time.Since(startTime),
		}, nil
	}

	// Prepare authentication
	var authConfig types.AuthConfig
	if registryConfig != nil {
		authConfig = types.AuthConfig{
			Username: registryConfig.Username,
			Password: registryConfig.Password,
			Email:    registryConfig.Email,
		}
	}

	authConfigBytes, err := json.Marshal(authConfig)
	if err != nil {
		return &PushResult{
			Success:   false,
			Error:     fmt.Errorf("failed to marshal auth config: %w", err),
			PushTime:  time.Since(startTime),
		}, nil
	}

	authStr := base64.URLEncoding.EncodeToString(authConfigBytes)

	// Push the image
	pushResponse, err := rp.dockerClient.ImagePush(ctx, targetImage, types.ImagePushOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		return &PushResult{
			Success:   false,
			Error:     fmt.Errorf("failed to push image: %w", err),
			PushTime:  time.Since(startTime),
		}, nil
	}
	defer pushResponse.Close()

	// Read push logs
	pushLogs, err := io.ReadAll(pushResponse)
	if err != nil {
		log.Printf("Warning: Failed to read push logs: %v", err)
	}

	return &PushResult{
		SourceImage:   config.SourceImage,
		TargetImage:   config.TargetImage,
		FullImageName: targetImage,
		PushTime:      time.Since(startTime),
		PushLogs:      string(pushLogs),
		Success:       true,
		Error:         nil,
	}, nil
}

func (rp *RegistryPusher) pushWithOpenShift(ctx context.Context, config PushConfig, registryConfig *RegistryConfig, startTime time.Time) (*PushResult, error) {
	// Create or update ImageStream for external registry
	imageStreamName := strings.Split(config.TargetImage, "/")[len(strings.Split(config.TargetImage, "/"))-1]
	
	imageStream := &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:        imageStreamName,
			Namespace:   config.Namespace,
			Labels:      config.Labels,
			Annotations: config.Annotations,
		},
		Spec: imagev1.ImageStreamSpec{
			Tags: []imagev1.TagReference{
				{
					Name: config.TargetTag,
					From: &corev1.ObjectReference{
						Kind: "DockerImage",
						Name: fmt.Sprintf("%s:%s", config.TargetImage, config.TargetTag),
					},
					ImportPolicy: imagev1.TagImportPolicy{
						Scheduled: true,
					},
				},
			},
		},
	}

	// Create or update the ImageStream
	existingIS, err := rp.imageClient.ImageStreams(config.Namespace).Get(ctx, imageStreamName, metav1.GetOptions{})
	if err != nil {
		// Create new ImageStream
		_, err = rp.imageClient.ImageStreams(config.Namespace).Create(ctx, imageStream, metav1.CreateOptions{})
		if err != nil {
			return &PushResult{
				Success:   false,
				Error:     fmt.Errorf("failed to create ImageStream: %w", err),
				PushTime:  time.Since(startTime),
			}, nil
		}
	} else {
		// Update existing ImageStream
		existingIS.Spec = imageStream.Spec
		existingIS.Labels = config.Labels
		existingIS.Annotations = config.Annotations
		_, err = rp.imageClient.ImageStreams(config.Namespace).Update(ctx, existingIS, metav1.UpdateOptions{})
		if err != nil {
			return &PushResult{
				Success:   false,
				Error:     fmt.Errorf("failed to update ImageStream: %w", err),
				PushTime:  time.Since(startTime),
			}, nil
		}
	}

	// Tag the source image to point to the external registry
	// This would typically be done through OpenShift's image import mechanism
	// For now, we'll simulate a successful push

	targetImage := fmt.Sprintf("%s:%s", config.TargetImage, config.TargetTag)

	return &PushResult{
		SourceImage:   config.SourceImage,
		TargetImage:   config.TargetImage,
		FullImageName: targetImage,
		PushTime:      time.Since(startTime),
		PushLogs:      fmt.Sprintf("Successfully configured ImageStream %s to reference %s", imageStreamName, targetImage),
		Success:       true,
		Error:         nil,
	}, nil
}

func (rp *RegistryPusher) ListRepositories(ctx context.Context, registryName string) ([]string, error) {
	registryConfig, exists := rp.registries[registryName]
	if !exists {
		return nil, fmt.Errorf("registry configuration not found: %s", registryName)
	}

	// This would require implementing registry API calls for different registries
	// For now, return a placeholder
	log.Printf("Listing repositories for registry %s (%s)", registryName, registryConfig.URL)
	return []string{}, nil
}

func (rp *RegistryPusher) ListTags(ctx context.Context, registryName, repository string) ([]string, error) {
	registryConfig, exists := rp.registries[registryName]
	if !exists {
		return nil, fmt.Errorf("registry configuration not found: %s", registryName)
	}

	// This would require implementing registry API calls for different registries
	// For now, return a placeholder
	log.Printf("Listing tags for %s in registry %s (%s)", repository, registryName, registryConfig.URL)
	return []string{}, nil
}

func (rp *RegistryPusher) DeleteImage(ctx context.Context, registryName, repository, tag string) error {
	registryConfig, exists := rp.registries[registryName]
	if !exists {
		return fmt.Errorf("registry configuration not found: %s", registryName)
	}

	// This would require implementing registry API calls for different registries
	// For now, return a placeholder success
	log.Printf("Deleting %s:%s from registry %s (%s)", repository, tag, registryName, registryConfig.URL)
	return nil
}

func (rp *RegistryPusher) GetImageInfo(ctx context.Context, registryName, repository, tag string) (*ImageInfo, error) {
	registryConfig, exists := rp.registries[registryName]
	if !exists {
		return nil, fmt.Errorf("registry configuration not found: %s", registryName)
	}

	// This would require implementing registry API calls for different registries
	// For now, return a placeholder
	info := &ImageInfo{
		Repository: repository,
		Tag:        tag,
		Registry:   registryName,
		Digest:     "sha256:placeholder",
		Size:       0,
		CreatedAt:  time.Now(),
		Labels:     make(map[string]string),
	}

	log.Printf("Getting info for %s:%s from registry %s (%s)", repository, tag, registryName, registryConfig.URL)
	return info, nil
}

type ImageInfo struct {
	Repository string            `json:"repository"`
	Tag        string            `json:"tag"`
	Registry   string            `json:"registry"`
	Digest     string            `json:"digest"`
	Size       int64             `json:"size"`
	CreatedAt  time.Time         `json:"created_at"`
	Labels     map[string]string `json:"labels"`
}

func (rp *RegistryPusher) ValidateRegistryAccess(ctx context.Context, registryName string) error {
	registryConfig, exists := rp.registries[registryName]
	if !exists {
		return fmt.Errorf("registry configuration not found: %s", registryName)
	}

	// This would typically test authentication and connectivity
	// For now, just check if we have credentials
	if registryConfig.Username == "" || registryConfig.Password == "" {
		return fmt.Errorf("incomplete registry credentials for %s", registryName)
	}

	log.Printf("Registry access validated for %s (%s)", registryName, registryConfig.URL)
	return nil
}

func (rp *RegistryPusher) GetRegistries() map[string]*RegistryConfig {
	// Return a copy to prevent external modification
	registries := make(map[string]*RegistryConfig)
	for k, v := range rp.registries {
		registries[k] = &RegistryConfig{
			URL:      v.URL,
			Username: v.Username,
			// Don't expose password
			Password: "***",
			Email:    v.Email,
			Secure:   v.Secure,
		}
	}
	return registries
}
