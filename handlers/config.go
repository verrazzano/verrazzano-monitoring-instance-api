// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package handlers

import (
	"flag"
	"net"
	"os"
	"strings"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	restgo "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetConfig returns the config for the monitoring instance API
func GetConfig() *restgo.Config {
	// create log for config
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Config").Str("name", "GetConfig").Logger()

	logger.Debug().Msgf("firing up at debug level: %d\n", debugLevel) // log at default debugLevel to ensure this line comes out

	flag.StringVar(&ListenURL, "ListenURL", ":9097", "set Cirith listener URL, default :9097")
	flag.StringVar(&promtoolPath, "promtoolPath", "/opt/tools/bin/promtool", "set path of promtool")
	flag.StringVar(&staticPath, "staticPath", "/usr/local/bin/static", "set path to static assets (e.g. Swagger)")
	flag.IntVar(&debugLevel, "debugLevel", LevelInfo, "debug level, 1 for most, 3 for least, 2 default."+
		"Setting a level lower than the default is not recommended in production.")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&vmiName, "vminame", "", "The name of the Verrazzano Monitoring Instance (VMI) object to manage")
	flag.StringVar(&namespace, "namespace", "default", "The namespace of the VMI object to manage")
	flag.Int64Var(&defaultMaxSize, "defaultMaxSize", 2000, "The default maximum size for any PVC in GB")
	flag.Int64Var(&defaultMinSize, "defaultMinSize", 50, "The default minimum size for any PVC in GB")
	var natGatewayIPsString string
	flag.StringVar(&natGatewayIPsString, "natGatewayIPs", "", "Comma-separated list of NAT Gateway IPs associated with this Verrazzano Monitoring Instance (VMI)'s environment")
	flag.StringVar(&ociConfigFile, "ociConfigFile", "", "Path to OCI config file.  Only required if out-of-cluster")
	flag.StringVar(&backupBucket, "backupBucket", "", "The name of Object Store bucket used to hold backups")
	flag.Parse()

	//Initialize the CFG
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logger.Fatal().Msgf("error building config: %+v", err)
	}
	s := schema.GroupVersion{Group: VMIGroup, Version: VMIVersion}
	cfg.GroupVersion = &s
	cfg.APIPath = "/apis"
	cfg.ContentType = runtime.ContentTypeJSON
	cfg.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: serializer.NewCodecFactory(&runtime.Scheme{})}

	// Initialize K8s client
	if vmiName == "" {
		vmiName = os.Getenv("VMI_NAME")
		if vmiName == "" {
			logger.Fatal().Msg("No Verrazzano Monitoring Instance (VMI) name set")
		}
	}
	if os.Getenv("NAMESPACE") != "" {
		namespace = os.Getenv("NAMESPACE")
	}

	// Parse and verify any NAT Gateway IPs specified
	natGatewayIPs = []net.IP{}
	if natGatewayIPsString != "" {
		for _, ipString := range strings.Split(natGatewayIPsString, ",") {
			ip := net.ParseIP(ipString)
			if ip == nil {
				logger.Fatal().Msgf("Invalid NAT Gateway IP: %s", ipString)
			}
			natGatewayIPs = append(natGatewayIPs, ip)
		}
	}

	cfg, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		logger.Fatal().Msgf("Error building kubeconfig: %s", err.Error())
	}

	if debugLevel < LevelMin {
		debugLevel = LevelMin
	}
	if debugLevel > LevelMax {
		debugLevel = LevelMax
	}

	if len(ociConfigFile) > 0 {
		// If ociConfigFile flag is passed, it must point to an existing readable file
		if _, err := os.Stat(ociConfigFile); err != nil {
			logger.Fatal().Msgf("OCI config %s does not exist, or cannot be read: %+v", ociConfigFile, err)
		}
	}

	logger.Info().Msgf("command line arguments: %v", os.Args[1:])

	logger.Debug().Msgf("Running at debug level: %d\n", debugLevel)

	return cfg
}
