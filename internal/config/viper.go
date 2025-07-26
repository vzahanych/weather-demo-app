package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

func Load(configPath string) (*Config, error) {
	cfg := NewDefaultConfig()

	v := viper.New()

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
	}

	v.SetEnvPrefix("WDP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	SetDefaultsFromStructRecursive(reflect.ValueOf(cfg), "", v)

	v.AutomaticEnv()

	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return cfg, nil
}

func SetDefaultsFromStructRecursive(v reflect.Value, prefix string, viper *viper.Viper) {
	// Handle pointer to struct
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		key := field.Tag.Get("mapstructure")
		if key == "" {
			key = strings.ToLower(field.Name)
		}

		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if fieldValue.Kind() == reflect.Struct {
			SetDefaultsFromStructRecursive(fieldValue, fullKey, viper)
		} else {
			viper.SetDefault(fullKey, fieldValue.Interface())
		}
	}
}
