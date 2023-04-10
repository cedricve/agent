package main

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/kerberos-io/agent/machinery/src/components"
	"github.com/kerberos-io/agent/machinery/src/log"
	"github.com/kerberos-io/agent/machinery/src/models"
	"github.com/kerberos-io/agent/machinery/src/routers"
	"github.com/kerberos-io/agent/machinery/src/utils"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

var VERSION = "3.0.0"

func main() {

	// You might be interested in debugging the agent.
	if os.Getenv("DATADOG_AGENT_ENABLED") == "true" {
		if os.Getenv("DATADOG_AGENT_K8S_ENABLED") == "true" {
			tracer.Start()
			defer tracer.Stop()
		} else {
			service := os.Getenv("DATADOG_AGENT_SERVICE")
			environment := os.Getenv("DATADOG_AGENT_ENVIRONMENT")
			log.Log.Info("Starting Datadog Agent with service: " + service + " and environment: " + environment)
			rules := []tracer.SamplingRule{tracer.RateRule(1)}
			tracer.Start(
				tracer.WithSamplingRules(rules),
				tracer.WithService(service),
				tracer.WithEnv(environment),
			)
			defer tracer.Stop()
			err := profiler.Start(
				profiler.WithService(service),
				profiler.WithEnv(environment),
				profiler.WithProfileTypes(
					profiler.CPUProfile,
					profiler.HeapProfile,
				),
			)
			if err != nil {
				log.Log.Fatal(err.Error())
			}
			defer profiler.Stop()
		}
	}

	// Start the show ;)
	action := os.Args[1]

	timezone, _ := time.LoadLocation("CET")
	log.Log.Init(timezone)

	switch action {

	case "version":
		log.Log.Info("You are currrently running Kerberos Agent " + VERSION)

		// same key and initialization vector as in ruby example
		key := []byte("7676BE0BA5945E52C37F13C8A5B2998DC9FE96F2E47D1B251B5B591B68C86BBE")
		iv := []byte("AF92C042E02A2FF88C939932AE342C90")

		// Initialize new crypter struct. Errors are ignored.
		crypter, _ := utils.NewCrypter(key, iv)

		// Lets encode plaintext using the same key and iv.
		// This will produce the very same result: "RanFyUZSP9u/HLZjyI5zXQ=="

		// Open file /Users/cedricverstraeten/Downloads/1681022365_6-967003_yolo23_200-200-400-400_0_769.mp4
		// and encode it to base64

		encrypted := "/Users/cedricverstraeten/Downloads/file.mp4"
		decrypted := "/Users/cedricverstraeten/Downloads/file_decry.mp4"
		f, _ := os.Open(encrypted)
		defer f.Close()
		bb, _ := ioutil.ReadAll(f)

		// Decode previous result. Should print "hello world"
		decoded, _ := crypter.DecryptECB(bb)
		fd, _ := os.Create(decrypted)
		defer fd.Close()
		fd.Write(decoded)

	case "discover":
		timeout := os.Args[2]
		log.Log.Info(timeout)

	case "run":
		{
			name := os.Args[2]
			port := os.Args[3]

			// Print Kerberos.io ASCII art
			utils.PrintASCIIArt()

			// Print the environment variables which include "AGENT_" as prefix.
			utils.PrintEnvironmentVariables()

			// Read the config on start, and pass it to the other
			// function and features. Please note that this might be changed
			// when saving or updating the configuration through the REST api or MQTT handler.
			var configuration models.Configuration
			configuration.Name = name
			configuration.Port = port

			// Open this configuration either from Kerberos Agent or Kerberos Factory.
			components.OpenConfig(&configuration)

			// We will override the configuration with the environment variables
			components.OverrideWithEnvironmentVariables(&configuration)

			// Printing final configuration
			utils.PrintConfiguration(&configuration)

			// Check the folder permissions, it might be that we do not have permissions to write
			// recordings, update the configuration or save snapshots.
			utils.CheckDataDirectoryPermissions()

			// Set timezone
			timezone, _ := time.LoadLocation(configuration.Config.Timezone)
			log.Log.Init(timezone)

			// Check if we have a device Key or not, if not
			// we will generate one.
			if configuration.Config.Key == "" {
				key := utils.RandStringBytesMaskImpr(30)
				configuration.Config.Key = key
				err := components.StoreConfig(configuration.Config)
				if err == nil {
					log.Log.Info("Main: updated unique key for agent to: " + key)
				} else {
					log.Log.Info("Main: something went wrong while trying to store key: " + key)
				}
			}

			// Bootstrapping the agent
			communication := models.Communication{
				HandleBootstrap: make(chan string, 1),
			}
			go components.Bootstrap(&configuration, &communication)

			// Start the REST API.
			routers.StartWebserver(&configuration, &communication)
		}
	default:
		log.Log.Error("Main: Sorry I don't understand :(")
	}
}
