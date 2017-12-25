package k8s

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/loomnetwork/dashboard/config"

	"strings"

	"github.com/pkg/errors"
)

var kubeConfigPath string
var gwi GatewayInstaller

const slug = "hello-world"

func TestInstallAndUpdate(t *testing.T) {
	c := &config.Config{KubeConfigPath: kubeConfigPath}

	t.Run("Install without a valid docker image should raise an error", func(t *testing.T) {
		err := Install(Gateway, slug, map[string]interface{}{"a": 1}, c)
		if err == nil {
			t.Fatal("Should have raised an Error")
			return
		}

		if !strings.Contains(err.Error(), "Config has no gateway image defined") {
			t.Errorf("Should complain about missing Image. Got %v instead", err.Error())
			return
		}

	})

	// Set the Image Path.
	c.GatewayDockerImage = "gcr.io/robotic-catwalk-188706/rpc_gateway:6fa56b0"

	t.Run("Install a gateway and wait for service, deployment and ingress", func(t *testing.T) {
		if err := Install(Gateway, slug, map[string]interface{}{"a": 1}, c); err != nil {
			t.Fatal(err)
			return
		}

		if err := assertDeploymentExists(slug); err != nil {
			t.Error(err)
			return
		}

		if err := assertServiceExists(slug); err != nil {
			t.Error(err)
			return
		}

		if err := assertIngressExists(slug); err != nil {
			t.Error(err)
			return
		}

		if err := Install(Gateway, slug, map[string]interface{}{"a": 1}, c); err != nil {
			t.Error("Gateway installation failed: ", err)
		}

		if err := assertDeploymentExists(slug); err != nil {
			t.Error(err)
			return
		}

		if err := assertServiceExists(slug); err != nil {
			t.Error(err)
			return
		}

		if err := assertIngressExists(slug); err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("Updating a few components should update the k8s resourece", func(t *testing.T) {
		newEnv := map[string]interface{}{
			"SPAWN_NETWORK":         "node /src/build/cli.node.js",
			"APP_ZIP_FILE":          "https://storage.googleapis.com/loomnetwork/block_ssh.zip",
			"DEMO_MODE":             "false",
			"PRIVATE_KEY_JSON_PATH": "data.json",
			"APP_SLUG":              slug,
		}

		//update setupO
		if err := Install(Gateway, slug, newEnv, c); err != nil {
			t.Errorf("Gateway updation failed: %v", err)
		}

		if err := assertDeploymentUpdated(slug, c, newEnv); err != nil {
			t.Error(err)
		}
	})
}

func TestMain(m *testing.M) {
	flag.StringVar(&kubeConfigPath, "kubeconfig", "", "Path to Kubernetes config.")
	if !flag.Parsed() {
		flag.Parse()
	}

	if kubeConfigPath == "" {
		log.Println("Missing -kubeconfig")
		os.Exit(127)
	}

	gwi = GatewayInstaller{}
	m.Run()
}

func assertDeploymentExists(slug string) error {
	cfg := &config.Config{KubeConfigPath: kubeConfigPath}
	client, err := makeClient(cfg)
	if err != nil {
		return err
	}

	d, err := gwi.getDeployment(makeGatewayName(slug), client)
	if err != nil {
		return errors.Errorf("Cannot get deployment: %v", err)
	}

	if expected := fmt.Sprintf("%v-%v", Gateway, slug); expected != d.ObjectMeta.GetName() {
		return errors.Errorf("Expected: %s \nActual: %s", expected, d.ObjectMeta.GetName())
	}

	return nil
}

func assertServiceExists(slug string) error {
	cfg := &config.Config{KubeConfigPath: kubeConfigPath}
	client, err := makeClient(cfg)
	if err != nil {
		return err
	}

	s, err := gwi.getService(makeGatewayName(slug), client)
	if err != nil {
		return errors.Errorf("Cannot get service: %v", err)
	}

	if expected := fmt.Sprintf("%v-%v", Gateway, slug); expected != s.ObjectMeta.GetName() {
		return errors.Errorf("Expected: %s \nActual: %s", expected, s.ObjectMeta.GetName())
	}

	return nil
}

func assertIngressExists(slug string) error {
	cfg := &config.Config{KubeConfigPath: kubeConfigPath}
	client, err := makeClient(cfg)
	if err != nil {
		return err
	}

	i, err := gwi.getIngress(makeIngressName(slug), client)
	if err != nil {
		return errors.Errorf("Cannot get ingress: %v", err)
	}

	if expected := makeIngressName(slug); expected != i.ObjectMeta.GetName() {
		return errors.Errorf("Expected: %s \nActual: %s", expected, i.ObjectMeta.GetName())
	}

	return nil
}

func assertDeploymentUpdated(slug string, cfg *config.Config, env map[string]interface{}) error {
	client, err := makeClient(cfg)
	if err != nil {
		return err
	}

	d, err := gwi.getDeployment(makeGatewayName(slug), client)
	if err != nil {
		return errors.Errorf("Cannot get deployment: %v", err)
	}

	expectedEnv := makeEnv(env)
	if !reflect.DeepEqual(d.Spec.Template.Spec.Containers[0].Env[0], expectedEnv) {
		errors.Errorf("Expected: %s Actual: %s", expectedEnv, d.Spec.Template.Spec.Containers[0].Env[0])
	}

	return nil
}
