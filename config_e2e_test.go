package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// These tests use a test harness so we can test running that harness
// with various command line args, environment variables and config
// files

// By default, use the valid config file
func setup(t *testing.T) string {
	return setupWithConfigFile(t, "config/config.yaml")
}

// Enable callers to specify the config file that they want to use
func setupWithConfigFile(t *testing.T, configFile string) string {
	os.Chdir("test")
	os.Setenv("CONFIG_A", "E")
	os.Setenv("CONFIG_B", "E")
	os.Setenv("CONFIG_SUB__G", "E")
	configString := fmt.Sprintf("--config=%s", configFile)
	cmd := exec.Command("./test", "-a=C", "--sub__h=C", configString)
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("You must compile the configuration test harness: cd test && go build:", err)
	}
	os.Chdir("..")
	return string(out)
}

// TestUnitNew tests that each of the config sources are hooked up properly
func TestUnitNew(t *testing.T) {
	conf := setup(t)

	if !strings.Contains(conf, "A: C") {
		t.Errorf("Command line args are not being set properly: %v", conf)
	}
	if !strings.Contains(conf, "B: E") {
		t.Errorf("Environment variables are not being set properly: %v", conf)
	}
	if !strings.Contains(conf, "C: O") {
		t.Errorf("Config file are not being set properly: %v", conf)
	}
	// if !strings.Contains(conf, "D: AL") {
	//     t.Errorf("App local are not being set properly: %v", conf)
	// }
	// if !strings.Contains(conf, "E: AD") {
	//     t.Errorf("App defaults are not being set properly: %v", conf)
	// }
	if !strings.Contains(conf, "Sub.G: E") {
		t.Errorf("Environment defaults are not being set properly: %v", conf)
	}
	if !strings.Contains(conf, "Sub.H: C") {
		t.Errorf("Command line args are not being set properly: %v", conf)
	}
	if !strings.Contains(conf, "L.A: 1") {
		t.Errorf("Lists are not being set properly: %v", conf)
	}
	if !strings.Contains(conf, "deep.deeper.deepest: x") {
		t.Errorf("Deeply nested file config is not being set properly: %v", conf)
	}
}

func TestUnitProvidedConfig(t *testing.T) {

	var configTests = []struct {
		name   string
		argsIn []string
	}{
		{"--=", []string{"-a=x", "--b=y", "--d", "w", "--config=something", "-e=u"}},
		{"--space", []string{"-a=x", "--b=y", "--d", "w", "--config", "something", "-e=u"}},
		{"--=", []string{"-a=x", "--b=y", "-c", "something", "--d", "w", "-e=u"}},
		{"--space", []string{"-a=x", "--b=y", "-c=something", "--d", "w", "-e=u"}},
	}

	for _, tt := range configTests {
		os.Args = tt.argsIn
		path := getConfigURI()
		if path != "something" {
			t.Errorf("The command line config file config is not working (%s): %v", tt.name, os.Args)
		}
	}
}
