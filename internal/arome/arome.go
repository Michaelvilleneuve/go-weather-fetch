package arome

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type AromeField struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Unit string `yaml:"unit"`
	Description string `yaml:"description"`
}

type AromeLayer struct {
	CommonName string `yaml:"name"`
	Fields []AromeField `yaml:"fields"`
}

type AromePackage struct {
	Name string `yaml:"name"`
	Layers []AromeLayer `yaml:"layers"`
}

type Arome struct {
	Packages []AromePackage `yaml:"packages"`
}

func (arome Arome) GetLayers() []AromeLayer {
	layers := []AromeLayer{}
	for _, aromePackage := range arome.Packages {
		layers = append(layers, aromePackage.Layers...)
	}
	return layers
}

func (arome Arome) GetLayerNames() []string {
	layers := []string{}
	for _, aromePackage := range arome.Packages {
		for _, aromeLayer := range aromePackage.Layers {
			layers = append(layers, aromeLayer.CommonName)
		}
	}
	return layers
}

func (aromePackage AromePackage) GetLayerNames() []string {
	layers := []string{}
	for _, aromeLayer := range aromePackage.Layers {
		layers = append(layers, aromeLayer.CommonName)
	}
	return layers
}

func (layer AromeLayer) GetFieldsNames() []string {
	fields := []string{}
	for _, field := range layer.Fields {
		fields = append(fields, field.Name)
	}
	return fields
}

func Configuration() Arome {
	yamlFile, err := os.ReadFile("config/arome.yml")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	var config Arome
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Error unmarshalling config file: %v", err)
	}

	return config
}
