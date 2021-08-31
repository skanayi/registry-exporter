package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	logs "github.com/sirupsen/logrus"
)

type Exporter struct {
	Logger zerolog.Logger

	RegistryMetric *prometheus.Desc
}

//Define the metrics we wish to expose

func main() {

	exporter := NewExporter()
	prometheus.MustRegister(exporter)

	http.Handle("/metrics", promhttp.Handler())

	logs.Fatal(http.ListenAndServe(":9300", nil))

}

func NewExporter() *Exporter {
	fs, _ := os.Create("exporter.log")
	log.Logger = log.With().Caller().Logger().Output(fs)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("starting exporter")

	var registryMetric = prometheus.NewDesc("registry_health_metric", "registry health", []string{"registry_instance", "app"}, nil)

	return &Exporter{Logger: log.Logger, RegistryMetric: registryMetric}

}

func (collector *Exporter) Describe(ch chan<- *prometheus.Desc) {

	//Update this section with the each metric you create for a given collector

	ch <- collector.RegistryMetric

}

//Collect implements required collect function for all promehteus collectors
func (collector *Exporter) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	//Write latest value for each metric in the prometheus metric channel.
	//Note that you can pass CounterValue, GaugeValue, or UntypedValue types here.

	ch <- prometheus.MustNewConstMetric(collector.RegistryMetric, prometheus.GaugeValue, collector.CheckRegistry(ctx, os.Getenv("REGISTRY_HOST")), os.Getenv("REGISTRY_HOST"), "registry")

}

func (exporter *Exporter) CheckRegistry(ctx context.Context, registryURL string) float64 {
	exporter.Logger.Info().Msgf("checking docker registry %s", registryURL)

	//os.RemoveAll("images")
	err := os.MkdirAll("images", 0777)
	if err != nil {
		exporter.Logger.Err(err).Msg("error creating images directory")

		return 0.0

	}

	fs, err := os.Create("images/Dockerfile")
	if err != nil {
		exporter.Logger.Err(err).Msg("error creating Dockerfile")

		return 0.0
	}

	defer fs.Close()
	bigBuff := make([]byte, 10000000)
	ioutil.WriteFile("images/dummy.test", bigBuff, 0666)
	fs.WriteString("FROM scratch \n")
	fs.WriteString("MAINTAINER sumesh \n")
	fs.WriteString("COPY dummy.test /dummy.test  \n")

	tar, err := archive.TarWithOptions("images/", &archive.TarOptions{})
	if err != nil {
		exporter.Logger.Err(err).Msg("error tar directory")

		return 0.0
	}
	defer tar.Close()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	exporter.Logger.Info().Msgf("docker client version is %s", dockerClient.ClientVersion())
	if err != nil {
		exporter.Logger.Err(err).Msg("error connecting to docker daemon")

		return 0.0

	}
	defer dockerClient.Close()

	authConfig := types.AuthConfig{}
	authConfig.Username = os.Getenv("REGISTRY_USERNAME")
	authConfig.Password = os.Getenv("REGISTRY_PASSWORF")

	authConfig.ServerAddress = os.Getenv("REGISTRY_HOST")
	authConfigBytes, _ := json.Marshal(authConfig)
	authConfigEncoded := base64.URLEncoding.EncodeToString(authConfigBytes)
	_, err = dockerClient.RegistryLogin(ctx, authConfig)

	if err != nil {
		exporter.Logger.Err(err).Msg("error logging in to to docker registry")
		return 0.0
	}

	opts := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{registryURL + "/devops/monitoring"},
		Remove:     true,
	}
	resp, err := dockerClient.ImageBuild(ctx, tar, opts)

	if err != nil {
		if err != nil {
			exporter.Logger.Err(err).Msg("error building docker images")

			return 0.0
		}

	}

	defer resp.Body.Close()

	dPush, err := dockerClient.ImagePush(ctx, registryURL+"/devops/monitoring", types.ImagePushOptions{RegistryAuth: authConfigEncoded})

	if err != nil {
		exporter.Logger.Err(err).Msg("error pushing docker images")

		return 0.0

	}
	defer dPush.Close()

	exporter.Logger.Info().Msg("checking registry completed")

	return 1.0
}
