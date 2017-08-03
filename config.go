// Package config loads configuration from multiple configuration sources.
// config.Load() should be all you need.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/go-yaml/yaml"
)

var (
	ConfigPrefix = "CONFIG"
	config       = make(map[interface{}]interface{})
	configMutex  = &sync.Mutex{}
	environment  = "dev"
	component    = ""
)

type Template struct {
	Search  string
	Replace string
}

var Templates []Template

// loadYAML converts the provided data to YAML and loads it into our
// global config. This can be called multiple times, each time will
// merge over previous values
func loadYAML(data []byte) error {
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}
	return nil
}

// getConfigURI pulls the config URI from the environment or from
// command line args
func getConfigURI() string {

	uri := ""

	// Pull uri from environment if it is set
	if envURI, ok := os.LookupEnv("CONFIG_URI"); ok {
		uri = envURI
	}

	// Pull uri from args, if it is present
	for _, pair := range parseCommandLineArgs() {
		if pair.Key == "config" || pair.Key == "c" {
			uri = pair.Val
			break
		}
	}

	return uri
}

// Load configuration, progressively:
// 1. Use the configuration data specified via --config or CONFIG_URI
// 2. Environment variables (":" or "__" as separator)
// 3. Command line args
func Load() error {

	if configURIS := getConfigURI(); configURIS != "" {

		// Split into individual URIs
		configs := strings.Split(configURIS, ";")

		for _, configURI := range configs {

			var (
				loader Loader
				err    error
			)

			if LoaderType(configURI) == "file" {
				loader, err = NewFileLoader(FileConfig{Path: configURI})
				if err != nil {
					return err
				}
			}

			if LoaderType(configURI) == "s3" {
				s3Config, err := S3ConfigFromURI(configURI)
				if err != nil {
					return err
				}
				loader, err = NewS3Loader(s3Config)
				if err != nil {
					return err
				}
			}

			if loader != nil {
				data, err := loader.Load()
				if err != nil {
					return err
				}

				if err := loadYAML(data); err != nil {
					return err
				}
			}
		}
	}

	// overwrite w/ env variables (starting with CONFIG_)
	loadEnvironmentVariables()

	// overwrite w/ command flags
	loadCommandLineArgs()

	// Set reserved config variables
	setEnvironment()

	return nil

}

// LoadComponent is a convenience func that sets the config component and Loads
func LoadComponent(comp string) {
	setComponent(comp)
	Load()
}

// setComponent is useful for giant configs that have multiple components
// specified in them (e.g. when all services are jammed together into a single
// file). In this case, we want to be able to nest an `env` section either at
// the top level, or within the specific component branch. If you call
// setComponent with an empty string, it will try to find the component in the
// existing config tree: this would have been set by any of the previous config
// methods; e.g. an `CONFIG_COMPONENT` env var or `--component=` cli flag
func setComponent(comp string) {

	// If something was passed in, use it
	if comp != "" {
		Set("comp", comp)
		component = comp
		return
	}

	// Otherwise, if it is in the environment, use it
	if comp := Get("comp"); comp != "" {
		component = comp
	}

}

// setEnvironment checks for the existence of an environment config
// If it finds it, it will use this environment for overrides
// during config reads. This would have been set by any of the previous
// config methods; e.g. an `CONFIG_ENV` env var or `--env=` cli flag
func setEnvironment() {
	if env := Get("env"); env != "" {
		environment = env
	} else {
		Set("env", environment)
	}
}

func Reset() {
	config = make(map[interface{}]interface{})
}

func nodes(key string) []string {
	return strings.Split(key, ":")
}

// make a local key out of an environment variable
// E.g., takes: CONFIG_FRIDGE__QUERY_SERVICE__FABRIC_ENDPOINT, or
//              config:fridge:query_service:fabric_endpoint
// and returns: config:fridge:query_service:fabric_endpoint
func normalizeKey(key string) string {
	key = strings.ToLower(key)
	key = strings.Replace(key, "__", ":", -1)
	return key
}

// stripConfigPrefix returns a string stripped of any of the different
// acceptable config prefixes. The second return value indicates whether
// or not the string has a config prefix
func stripConfigPrefix(s string) (string, bool) {
	configPrefixes := []string{
		fmt.Sprintf("%s__", ConfigPrefix),
		fmt.Sprintf("%s_", ConfigPrefix),
		fmt.Sprintf("%s:", ConfigPrefix),
	}
	compareString := strings.ToUpper(s)
	for _, prefix := range configPrefixes {
		if strings.HasPrefix(compareString, prefix) {
			return strings.TrimPrefix(compareString, prefix), true
		}
	}
	return s, false
}

// mkPath is a helper function to create the required nodes in the config tree
func mkPath(fullKey string) (map[interface{}]interface{}, interface{}) {

	fullKey = normalizeKey(fullKey)
	// nodes() spits up the string into its constituent parts, assumes
	// that ":" is used as a delimiter
	nodeValues := nodes(fullKey)

	// start at root map
	currentNode := config
	var key string

	if len(nodeValues) == 1 {
		return currentNode, nodeValues[0]
	}

	configMutex.Lock()
	defer configMutex.Unlock()
	for i, nodeValue := range nodeValues {

		// if this is the last element in the key,
		// we will break and return this value
		if i == len(nodeValues)-1 {
			key = nodeValue
			break
		}

		// if node map isn't created on this non-leaf, make one
		if _, ok := currentNode[nodeValue]; !ok {
			currentNode[nodeValue] = make(map[interface{}]interface{})
		}

		// move onto next node
		currentNode = currentNode[nodeValue].(map[interface{}]interface{})
	}

	return currentNode, key
}

// Set lets you set/override specific leaves of the config tree
func Set(keyPath string, value interface{}) {
	node, key := mkPath(keyPath)
	configMutex.Lock()
	node[key] = value
	configMutex.Unlock()
}

// SetJSON allows you to set an entire JSON string into the config
// If the provided json string is invalid, you will receive an error
func SetJSON(keyPath string, jsonString string) error {
	node, key := mkPath(keyPath)

	// Get the JSON
	var jsonData interface{}
	err := json.Unmarshal([]byte(jsonString), &jsonData)
	if err != nil {
		return err
	}

	// JSON is a subset of YAML. We use YAML as our config
	// data structure, so we need to convert this JSON into
	// YAML

	// Convert to YAML
	yamlString, _ := yaml.Marshal(jsonData)

	// Load the YAML
	var values interface{}
	yaml.Unmarshal(yamlString, &values)

	// Done. Let's set it on the node now
	configMutex.Lock()
	node[key] = values
	configMutex.Unlock()

	return nil
}

func SetList(key string, list string) {
	// TODO: parse list into a string array and set it
}

// GetAny returns whatever it finds at a specific config node
func GetAny(key string) interface{} {
	cfg := getEnvironmentedT(key)
	cfg = evalTemplatesAll(cfg)
	return cfg
}

// Get is the typical reader. It returns a value as a string
// e.g. Get("fridge:query_service:fabric_endpoint")
func Get(key string) string {
	return GetString(key)
}

// GetString is the typical reader. It returns a value as a string.
// e.g. Get("fridge:query_service:fabric_endpoint")
// If the specified key does not exist, an empty
// string is returned.
func GetString(key string) string {
	switch v := getEnvironmentedT(key).(type) {
	case string:
		return evalTemplate(v)
	case int:
		return strconv.Itoa(v)
	}
	return ""
}

// GetInt returns a value as an int if the
// specified key exists, 0 if the key does
// not exist
func GetInt(key string) int {
	switch v := getEnvironmentedT(key).(type) {
	case int:
		return v
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 0
}

// GetBool returns a value as a boolean if the
// specified key exists, false if the key does
// not exist
func GetBool(key string) bool {
	switch v := getEnvironmentedT(key).(type) {
	case bool:
		return v
	case string:
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return false
}

// getEnvironmentedT will return the component and non-component
// environment-overridden configuration value if it exists. Resolves more
// specific definitions first. The specific order is:
// component:<component>:env:<environment>, component:<component>, then
// env:<environment>, and finally, from the root
func getEnvironmentedT(key string) interface{} {
	// Order is important here.
	// We start at the top of the config
	// and walk our way down by overwriting the config values as we find them.

	// first get the default value (the top level value)
	val := getT(key)

	// now look for the  key in the environment
	envValue := getT(fmt.Sprintf("environment:%s:%s", environment, key))

	// If we found a value in the environemnt overwrite its fields with
	// the default one.
	if envValue != nil {
		val = merge(envValue, val)
	}

	// If we have a component lets write those values.
	if component != "" {

		// Look for key in top level of the component
		componentVal := getT(fmt.Sprintf("component:%s:%s", component, key))
		if componentVal != nil {
			val = merge(componentVal, val)
		}

		// Look for key in component's env
		componentEnvValue := getT(fmt.Sprintf("component:%s:environment:%s:%s", component, environment, key))
		if componentEnvValue != nil {
			val = merge(componentEnvValue, val)
		}
	}

	return val
}

// getT walks the node-tree rooted at the node stored in config.
// Returns the specified value if it is present, and nil if the
// key is not present.
// Also checks for environment overrides
func getT(key string) interface{} {

	key = strings.ToLower(key)

	// walk the requested nodes to get to the value
	nodeValues := nodes(key)
	currentNode := config
	var val interface{}
	var ok bool
	for i, nodeValue := range nodeValues {

		// if the next node doesn't exist, exit early
		val, ok = currentNode[nodeValue]
		if !ok {
			return nil
		}

		// if we are to a leaf, we've won
		if i == len(nodeValues)-1 {
			break
		}
		currentNode, ok = val.(map[interface{}]interface{})
		if !ok {
			// The next node isn't a map[string]interface{}... that
			// means the requested key-path is invalid
			return nil
		}
	}
	return val
}

// GetAll gives you access to the raw config var
// Useful for debugging
func GetAll() map[interface{}]interface{} {
	return config
}

// ToYAML returns the current config as a YAML doc
// Useful for debugging
func ToYAML() string {
	out, _ := yaml.Marshal(config)
	return string(out)
}

// ToGo returns a Go-syntax representation of the config
func ToGo() string {
	return fmt.Sprintf("%#v", config)
}

// evalTemplate replaces all templatized variables in the given string
// with their evaluated values
func evalTemplate(s string) string {
	for _, template := range Templates {
		s = strings.Replace(
			s,
			fmt.Sprintf("{%s}", template.Search),
			template.Replace,
			-1,
		)
	}
	return s
}

// evalTemplatesAll replaces all templatized variables in the given
// nested config
func evalTemplatesAll(cfg interface{}) interface{} {
	switch cfg := cfg.(type) {
	case bool, int:
		return cfg
	case string:
		return evalTemplate(cfg)
	case []interface{}:
		var _cfg []interface{}
		for _, item := range cfg {
			_cfg = append(_cfg, evalTemplatesAll(item))
		}
		return _cfg
	case map[interface{}]interface{}:
		_cfg := make(map[interface{}]interface{})
		for k, v := range cfg {
			_cfg[k] = evalTemplatesAll(v)
		}
		return _cfg
	}
	return cfg
}

// merge two maps.
// src values are used on both src and dst.
// if the values are not maps, src is returned.
func merge(srcAInterface, dstAsInterface interface{}) interface{} {
	src, ok := srcAInterface.(map[interface{}]interface{})
	if !ok {
		return srcAInterface
	}

	dst, ok := dstAsInterface.(map[interface{}]interface{})
	if !ok {
		return srcAInterface
	}

	for key, srcVal := range src {
		if dstVal, ok := dst[key]; ok {
			srcMap, srcMapOk := srcVal.(map[interface{}]interface{})
			dstMap, dstMapOk := dstVal.(map[interface{}]interface{})
			if srcMapOk && dstMapOk {
				srcVal = merge(dstMap, srcMap)
			}
		}
		dst[key] = srcVal
	}
	return dst
}
