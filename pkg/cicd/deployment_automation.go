package cicd

import (
	"context"
	"fmt"
	"log"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	// Networking for Ingress support
	networkingv1 "k8s.io/api/networking/v1"
)

type DeploymentAutomation struct {
	kubeClient      kubernetes.Interface
	kubeConfig      *rest.Config
	defaultTemplate *DeploymentTemplate
}

type DeploymentConfig struct {
	Name          string
	Namespace     string
	Image         string
	Tag           string
	Replicas      int32
	Port          int32
	ServiceType   string
	Labels        map[string]string
	Annotations   map[string]string
	EnvVars       map[string]string
	Resources     *ResourceRequirements
	Strategy      string // "recreate", "rolling", "blue-green"
	ExposeIngress bool
	IngressDomain string
}

type ResourceRequirements struct {
	Requests map[string]string
	Limits   map[string]string
}

type DeploymentTemplate struct {
	DefaultReplicas  int32
	DefaultPort      int32
	DefaultResources *ResourceRequirements
	DefaultLabels    map[string]string
	DefaultEnvVars   map[string]string
}

type DeploymentResult struct {
	Name        string
	Namespace   string
	Image       string
	Status      string
	Replicas    string
	ServiceName string
	IngressURL  string
	DeployTime  time.Duration
	Success     bool
	Error       error
	Logs        []string
}

func NewDeploymentAutomation(kubeConfig *rest.Config) (*DeploymentAutomation, error) {
	// Create Kubernetes client
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Set default template
	defaultTemplate := &DeploymentTemplate{
		DefaultReplicas: 1,
		DefaultPort:     8080,
		DefaultResources: &ResourceRequirements{
			Requests: map[string]string{
				"memory": "256Mi",
				"cpu":    "100m",
			},
			Limits: map[string]string{
				"memory": "512Mi",
				"cpu":    "500m",
			},
		},
		DefaultLabels: map[string]string{
			"app.kubernetes.io/managed-by": "ai-mcp-openshift-server",
		},
		DefaultEnvVars: map[string]string{
			"PORT": "8080",
		},
	}

	return &DeploymentAutomation{
		kubeClient:      kubeClient,
		kubeConfig:      kubeConfig,
		defaultTemplate: defaultTemplate,
	}, nil
}

func (da *DeploymentAutomation) DeployApplication(ctx context.Context, config DeploymentConfig) (*DeploymentResult, error) {
	startTime := time.Now()
	logs := []string{}

	// Apply defaults
	if config.Replicas == 0 {
		config.Replicas = da.defaultTemplate.DefaultReplicas
	}
	if config.Port == 0 {
		config.Port = da.defaultTemplate.DefaultPort
	}
	if config.Resources == nil {
		config.Resources = da.defaultTemplate.DefaultResources
	}
	if config.Labels == nil {
		config.Labels = make(map[string]string)
	}
	for k, v := range da.defaultTemplate.DefaultLabels {
		if _, exists := config.Labels[k]; !exists {
			config.Labels[k] = v
		}
	}
	if config.EnvVars == nil {
		config.EnvVars = make(map[string]string)
	}
	for k, v := range da.defaultTemplate.DefaultEnvVars {
		if _, exists := config.EnvVars[k]; !exists {
			config.EnvVars[k] = v
		}
	}

	// Ensure namespace exists
	if err := da.ensureNamespace(ctx, config.Namespace); err != nil {
		return &DeploymentResult{
			Success:    false,
			Error:      fmt.Errorf("failed to ensure namespace: %w", err),
			DeployTime: time.Since(startTime),
			Logs:       logs,
		}, nil
	}
	logs = append(logs, fmt.Sprintf("Ensured namespace %s exists", config.Namespace))

	// Create or update deployment
	deployment, err := da.createOrUpdateDeployment(ctx, config)
	if err != nil {
		return &DeploymentResult{
			Success:    false,
			Error:      fmt.Errorf("failed to create/update deployment: %w", err),
			DeployTime: time.Since(startTime),
			Logs:       logs,
		}, nil
	}
	logs = append(logs, fmt.Sprintf("Created/updated deployment %s", deployment.Name))

	// Create or update service
	service, err := da.createOrUpdateService(ctx, config)
	if err != nil {
		return &DeploymentResult{
			Success:    false,
			Error:      fmt.Errorf("failed to create/update service: %w", err),
			DeployTime: time.Since(startTime),
			Logs:       logs,
		}, nil
	}
	logs = append(logs, fmt.Sprintf("Created/updated service %s", service.Name))

	// Create ingress if requested
	var ingressURL string
	if config.ExposeIngress {
		ingress, err := da.createOrUpdateIngress(ctx, config)
		if err != nil {
			log.Printf("Warning: Failed to create ingress: %v", err)
			logs = append(logs, fmt.Sprintf("Warning: Failed to create ingress: %v", err))
		} else {
			if len(ingress.Spec.Rules) > 0 {
				ingressURL = fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
				logs = append(logs, fmt.Sprintf("Created/updated ingress %s with URL %s", ingress.Name, ingressURL))
			}
		}
	}

	// Wait for deployment to be ready
	if err := da.waitForDeployment(ctx, config.Namespace, config.Name, 5*time.Minute); err != nil {
		return &DeploymentResult{
			Success:    false,
			Error:      fmt.Errorf("deployment did not become ready: %w", err),
			DeployTime: time.Since(startTime),
			Logs:       logs,
		}, nil
	}
	logs = append(logs, fmt.Sprintf("Deployment %s is ready", config.Name))

	// Get final deployment status
	deployment, err = da.kubeClient.AppsV1().Deployments(config.Namespace).Get(ctx, config.Name, metav1.GetOptions{})
	if err != nil {
		return &DeploymentResult{
			Success:    false,
			Error:      fmt.Errorf("failed to get final deployment status: %w", err),
			DeployTime: time.Since(startTime),
			Logs:       logs,
		}, nil
	}

	return &DeploymentResult{
		Name:        config.Name,
		Namespace:   config.Namespace,
		Image:       fmt.Sprintf("%s:%s", config.Image, config.Tag),
		Status:      "Ready",
		Replicas:    fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, deployment.Status.Replicas),
		ServiceName: service.Name,
		IngressURL:  ingressURL,
		DeployTime:  time.Since(startTime),
		Success:     true,
		Error:       nil,
		Logs:        logs,
	}, nil
}

func (da *DeploymentAutomation) ensureNamespace(ctx context.Context, namespace string) error {
	_, err := da.kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		// Create namespace if it doesn't exist
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "ai-mcp-openshift-server",
				},
			},
		}
		_, err = da.kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	}
	return nil
}

func (da *DeploymentAutomation) createOrUpdateDeployment(ctx context.Context, config DeploymentConfig) (*appsv1.Deployment, error) {
	labels := config.Labels
	labels["app"] = config.Name
	labels["version"] = config.Tag

	// Prepare environment variables
	var envVars []corev1.EnvVar
	for k, v := range config.EnvVars {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	// Prepare resource requirements
	var resources corev1.ResourceRequirements
	if config.Resources != nil {
		resources = corev1.ResourceRequirements{
			Requests: make(corev1.ResourceList),
			Limits:   make(corev1.ResourceList),
		}
		for k, v := range config.Resources.Requests {
			resources.Requests[corev1.ResourceName(k)] = resource.MustParse(v)
		}
		for k, v := range config.Resources.Limits {
			resources.Limits[corev1.ResourceName(k)] = resource.MustParse(v)
		}
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        config.Name,
			Namespace:   config.Namespace,
			Labels:      labels,
			Annotations: config.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &config.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": config.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  config.Name,
							Image: fmt.Sprintf("%s:%s", config.Image, config.Tag),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: config.Port,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env:       envVars,
							Resources: resources,
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(int(config.Port)),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(int(config.Port)),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
						},
					},
				},
			},
		},
	}

	// Try to get existing deployment
	existingDeployment, err := da.kubeClient.AppsV1().Deployments(config.Namespace).Get(ctx, config.Name, metav1.GetOptions{})
	if err != nil {
		// Create new deployment
		return da.kubeClient.AppsV1().Deployments(config.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
	} else {
		// Update existing deployment
		deployment.ObjectMeta.ResourceVersion = existingDeployment.ObjectMeta.ResourceVersion
		return da.kubeClient.AppsV1().Deployments(config.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	}
}

func (da *DeploymentAutomation) createOrUpdateService(ctx context.Context, config DeploymentConfig) (*corev1.Service, error) {
	labels := config.Labels
	labels["app"] = config.Name

	serviceType := corev1.ServiceTypeClusterIP
	if config.ServiceType != "" {
		serviceType = corev1.ServiceType(config.ServiceType)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        config.Name,
			Namespace:   config.Namespace,
			Labels:      labels,
			Annotations: config.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Type: serviceType,
			Ports: []corev1.ServicePort{
				{
					Port:       config.Port,
					TargetPort: intstr.FromInt(int(config.Port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": config.Name,
			},
		},
	}

	// Try to get existing service
	existingService, err := da.kubeClient.CoreV1().Services(config.Namespace).Get(ctx, config.Name, metav1.GetOptions{})
	if err != nil {
		// Create new service
		return da.kubeClient.CoreV1().Services(config.Namespace).Create(ctx, service, metav1.CreateOptions{})
	} else {
		// Update existing service
		service.ObjectMeta.ResourceVersion = existingService.ObjectMeta.ResourceVersion
		service.Spec.ClusterIP = existingService.Spec.ClusterIP
		return da.kubeClient.CoreV1().Services(config.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	}
}

func (da *DeploymentAutomation) createOrUpdateIngress(ctx context.Context, config DeploymentConfig) (*networkingv1.Ingress, error) {
	labels := config.Labels
	labels["app"] = config.Name

	pathType := networkingv1.PathTypePrefix

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        config.Name,
			Namespace:   config.Namespace,
			Labels:      labels,
			Annotations: config.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: func() string {
						if config.IngressDomain != "" {
							return fmt.Sprintf("%s.%s", config.Name, config.IngressDomain)
						}
						return fmt.Sprintf("%s.example.com", config.Name)
					}(),
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: config.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: config.Port,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Try to get existing ingress
	existingIngress, err := da.kubeClient.NetworkingV1().Ingresses(config.Namespace).Get(ctx, config.Name, metav1.GetOptions{})
	if err != nil {
		// Create new ingress
		return da.kubeClient.NetworkingV1().Ingresses(config.Namespace).Create(ctx, ingress, metav1.CreateOptions{})
	} else {
		// Update existing ingress
		ingress.ObjectMeta.ResourceVersion = existingIngress.ObjectMeta.ResourceVersion
		return da.kubeClient.NetworkingV1().Ingresses(config.Namespace).Update(ctx, ingress, metav1.UpdateOptions{})
	}
}

func (da *DeploymentAutomation) waitForDeployment(ctx context.Context, namespace, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			deployment, err := da.kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get deployment: %w", err)
			}

			if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas &&
				deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas &&
				deployment.Status.ObservedGeneration >= deployment.Generation {
				return nil
			}
		}
	}
}

func (da *DeploymentAutomation) DeleteApplication(ctx context.Context, namespace, name string) error {
	logs := []string{}

	// Delete deployment
	err := da.kubeClient.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logs = append(logs, fmt.Sprintf("Warning: Failed to delete deployment: %v", err))
	} else {
		logs = append(logs, fmt.Sprintf("Deleted deployment %s", name))
	}

	// Delete service
	err = da.kubeClient.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logs = append(logs, fmt.Sprintf("Warning: Failed to delete service: %v", err))
	} else {
		logs = append(logs, fmt.Sprintf("Deleted service %s", name))
	}

	// Delete ingress
	err = da.kubeClient.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logs = append(logs, fmt.Sprintf("Warning: Failed to delete ingress: %v", err))
	} else {
		logs = append(logs, fmt.Sprintf("Deleted ingress %s", name))
	}

	for _, logEntry := range logs {
		log.Println(logEntry)
	}

	return nil
}

func (da *DeploymentAutomation) ListApplications(ctx context.Context, namespace string) ([]ApplicationInfo, error) {
	deployments, err := da.kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/managed-by=ai-mcp-openshift-server",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var apps []ApplicationInfo
	for _, deployment := range deployments.Items {
		app := ApplicationInfo{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			Image:     deployment.Spec.Template.Spec.Containers[0].Image,
			Replicas:  fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, deployment.Status.Replicas),
			Status:    string(deployment.Status.Conditions[len(deployment.Status.Conditions)-1].Type),
			CreatedAt: deployment.CreationTimestamp.Time,
			Labels:    deployment.Labels,
		}

		// Try to get service info
		service, err := da.kubeClient.CoreV1().Services(namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
		if err == nil {
			app.ServiceName = service.Name
			if len(service.Spec.Ports) > 0 {
				app.Port = service.Spec.Ports[0].Port
			}
		}

		// Try to get ingress info
		ingress, err := da.kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
		if err == nil && len(ingress.Spec.Rules) > 0 {
			app.IngressURL = fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
		}

		apps = append(apps, app)
	}

	return apps, nil
}

type ApplicationInfo struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Image       string            `json:"image"`
	Replicas    string            `json:"replicas"`
	Status      string            `json:"status"`
	ServiceName string            `json:"service_name,omitempty"`
	Port        int32             `json:"port,omitempty"`
	IngressURL  string            `json:"ingress_url,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	Labels      map[string]string `json:"labels"`
}
