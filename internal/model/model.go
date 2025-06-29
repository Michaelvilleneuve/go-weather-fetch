package model

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Field struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Unit string `yaml:"unit"`
	Description string `yaml:"description"`
}

type Layer struct {
	CommonName string `yaml:"name"`
	Fields []Field `yaml:"fields"`
}

type Package struct {
	Name string `yaml:"name"`
	Layers []Layer `yaml:"layers"`
	Run *string
	Hour *string
}

type Model struct {
	Packages []Package `yaml:"packages"`
}


func (model Model) GetLayerNames() []string {
	layers := []string{}
	for _, modelPackage := range model.Packages {
		for _, layer := range modelPackage.Layers {
			layers = append(layers, layer.CommonName)
		}
	}
	return layers
}

func GetModel(modelName string) Model {
	yamlFile, err := os.ReadFile("config/" + modelName + ".yml")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	var config Model
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Error unmarshalling config file: %v", err)
	}

	return config
}