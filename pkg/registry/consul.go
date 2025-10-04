package registry

import (
	"fmt"
	"net"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/yourusername/im-system/pkg/logger"
	"go.uber.org/zap"
)

type ConsulRegistry struct {
	client         *consulapi.Client
	config         *consulapi.Config
	serviceID      string
	serviceName    string
	serviceAddress string
	servicePort    int
	checkInterval  time.Duration
	deregisterTime time.Duration
}

type ServiceConfig struct {
	Address        string
	Scheme         string
	ServiceName    string
	ServiceAddress string
	ServicePort    int
	CheckInterval  time.Duration
	DeregisterTime time.Duration
	Tags           []string
	Meta           map[string]string
}

// NewConsulRegistry creates a new Consul registry client
func NewConsulRegistry(cfg *ServiceConfig) (*ConsulRegistry, error) {
	config := consulapi.DefaultConfig()
	config.Address = cfg.Address
	config.Scheme = cfg.Scheme

	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	// Get local IP if service address is not provided
	serviceAddr := cfg.ServiceAddress
	if serviceAddr == "" || serviceAddr == "0.0.0.0" {
		serviceAddr, err = getLocalIP()
		if err != nil {
			return nil, fmt.Errorf("failed to get local IP: %w", err)
		}
	}

	serviceID := fmt.Sprintf("%s-%s-%d", cfg.ServiceName, serviceAddr, cfg.ServicePort)

	return &ConsulRegistry{
		client:         client,
		config:         config,
		serviceID:      serviceID,
		serviceName:    cfg.ServiceName,
		serviceAddress: serviceAddr,
		servicePort:    cfg.ServicePort,
		checkInterval:  cfg.CheckInterval,
		deregisterTime: cfg.DeregisterTime,
	}, nil
}

// Register registers the service with Consul
func (r *ConsulRegistry) Register(tags []string, meta map[string]string) error {
	registration := &consulapi.AgentServiceRegistration{
		ID:      r.serviceID,
		Name:    r.serviceName,
		Address: r.serviceAddress,
		Port:    r.servicePort,
		Tags:    tags,
		Meta:    meta,
		Check: &consulapi.AgentServiceCheck{
			CheckID:                        fmt.Sprintf("check-%s", r.serviceID),
			Name:                           fmt.Sprintf("Health check for %s", r.serviceName),
			TTL:                            r.checkInterval.String(),
			DeregisterCriticalServiceAfter: r.deregisterTime.String(),
		},
	}

	if err := r.client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	logger.Log.Info("Service registered with Consul",
		zap.String("service_id", r.serviceID),
		zap.String("service_name", r.serviceName),
		zap.String("address", r.serviceAddress),
		zap.Int("port", r.servicePort),
	)

	// Start health check heartbeat
	go r.healthCheckHeartbeat()

	return nil
}

// Deregister removes the service from Consul
func (r *ConsulRegistry) Deregister() error {
	if err := r.client.Agent().ServiceDeregister(r.serviceID); err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}

	logger.Log.Info("Service deregistered from Consul",
		zap.String("service_id", r.serviceID),
	)

	return nil
}

// DiscoverService discovers healthy instances of a service
func (r *ConsulRegistry) DiscoverService(serviceName string) ([]*consulapi.ServiceEntry, error) {
	services, _, err := r.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover service: %w", err)
	}

	return services, nil
}

// GetServiceAddress gets a random healthy service address
func (r *ConsulRegistry) GetServiceAddress(serviceName string) (string, error) {
	services, err := r.DiscoverService(serviceName)
	if err != nil {
		return "", err
	}

	if len(services) == 0 {
		return "", fmt.Errorf("no healthy instances found for service: %s", serviceName)
	}

	// Simple round-robin: return the first one (can be enhanced with load balancing)
	service := services[0]
	address := fmt.Sprintf("%s:%d", service.Service.Address, service.Service.Port)

	return address, nil
}

// healthCheckHeartbeat sends periodic health check updates to Consul
func (r *ConsulRegistry) healthCheckHeartbeat() {
	ticker := time.NewTicker(r.checkInterval / 2) // Send updates more frequently than check interval
	defer ticker.Stop()

	checkID := fmt.Sprintf("check-%s", r.serviceID)

	for range ticker.C {
		err := r.client.Agent().UpdateTTL(checkID, "Service is healthy", consulapi.HealthPassing)
		if err != nil {
			logger.Log.Error("Failed to update TTL",
				zap.String("service_id", r.serviceID),
				zap.Error(err),
			)
		}
	}
}

// getLocalIP gets the local non-loopback IP address
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no non-loopback IP address found")
}
