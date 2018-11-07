package integration

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/kubernetes-incubator/kube-aws/core/root"
	"github.com/kubernetes-incubator/kube-aws/core/root/config"
	"github.com/kubernetes-incubator/kube-aws/model"
	"github.com/kubernetes-incubator/kube-aws/plugin"
	"github.com/kubernetes-incubator/kube-aws/test/helper"
)

func TestPlugin(t *testing.T) {
	kubeAwsSettings := newKubeAwsSettingsFromEnv(t)

	s3URI, s3URIExists := os.LookupEnv("KUBE_AWS_S3_DIR_URI")

	if !s3URIExists || s3URI == "" {
		s3URI = "s3://mybucket/mydir"
		t.Logf(`Falling back s3URI to a stub value "%s" for tests of validating stack templates. No assets will actually be uploaded to S3`, s3URI)
	}

	minimalValidConfigYaml := kubeAwsSettings.minimumValidClusterYamlWithAZ("c")

	validCases := []struct {
		context       string
		clusterYaml   string
		plugins       []helper.TestPlugin
		assertConfig  []ConfigTester
		assertCluster []ClusterTester
	}{
		{
			context: "WithAddons",
			clusterYaml: minimalValidConfigYaml + `


kubeAwsPlugins:
  myPlugin:
    enabled: true
    queue:
      name: baz1
    oidc:
      issuer:
        url: "https://login.example.com/"

worker:
  nodePools:
  - name: pool1
    kubeAwsPlugins:
      myPlugin:
        enabled: true
        queue:
          name: baz2
`,
			plugins: []helper.TestPlugin{
				helper.TestPlugin{
					Name: "my-plugin",
					Files: map[string]string{
						"assets/controller/baz.txt": "controller-baz",
						"assets/etcd/baz.txt":       "etcd-baz",
						"assets/worker/baz.txt":     "worker-baz",
					},
					Yaml: `
metadata:
  name: my-plugin
  version: 0.0.1
spec:
  configuration:
    # This is the defaults for the values passed to templates like:
    # * cloudformation.stacks.{controlPlane,nodePool,root}.resources.append and
    # * kubernetes.apiserer.flags[].value
    #
    # The defaults can be overridden from cluster.yaml via:
    # * kubeAwsPlugins.pluginName.* and
    # * worker.nodePools[].kubeAwsPlugins.pluginName.*
    values:
      queue:
        name: bar
      oidc:
        issuer:
          url: unspecified
    cloudformation:
      stacks:
        controlPlane:
          resources:
            append:
              inline: |
                {
                  "QueueFromMyPlugin": {
                    "Type": "AWS::SQS::Queue",
                    "Properties": {
                      "QueueName": {{quote .Values.queue.name}}
                    }
                  }
                }
        nodePool:
          resources:
            append:
              inline: |
                {
                  "QueueFromMyPlugin": {
                    "Type": "AWS::SQS::Queue",
                    "Properties": {
                      "QueueName": {{quote .Values.queue.name}}
                    }
                  }
                }
        root:
          resources:
            append:
              inline: |
                {
                  "QueueFromMyPlugin": {
                    "Type": "AWS::SQS::Queue",
                    "Properties": {
                    "QueueName": {{quote .Values.queue.name}}
                    }
                  }
                }
        etcd:
          resources:
            append:
              inline: |
                {
                  "QueueFromMyPlugin": {
                    "Type": "AWS::SQS::Queue",
                    "Properties": {
                    "QueueName": {{quote .Values.queue.name}}
                    }
                  }
                }
        network:
          resources:
            append:
              inline: |
                {
                  "QueueFromMyPlugin": {
                    "Type": "AWS::SQS::Queue",
                    "Properties": {
                    "QueueName": {{quote .Values.queue.name}}
                    }
                  }
                }
    kubernetes:
      apiserver:
        flags:
        - name: "oidc-issuer-url"
          value: "{{ .Values.oidc.issuer.url}}"
        volumes:
        - name: "mycreds"
          path: "/etc/my/creds"
    node:
      roles:
        controller:
          iam:
            policy:
              statements:
              - actions:
                - "ec2:Describe*"
                effect: "Allow"
                resources:
                - "*"
          kubelet:
            nodeLabels:
              role: controller
          systemd:
            units:
            - name: save-queue-name.service
              contents:
                inline: |
                  [Unit]
          storage:
            files:
            - path: /var/kube-aws/bar.txt
              permissions: 0644
              contents:
                inline: controller-bar
            - path: /var/kube-aws/baz.txt
              permissions: 0644
              contents:
                source:
                  path: assets/controller/baz.txt
        etcd:
          iam:
            policy:
              statements:
              - actions:
                - "ec2:Describe*"
                effect: "Allow"
                resources:
                - "*"
          systemd:
            units:
            - name: save-queue-name.service
              contents:
                inline: |
                  [Unit]
          storage:
            files:
            - path: /var/kube-aws/bar.txt
              permissions: 0644
              contents:
                inline: etcd-bar
            - path: /var/kube-aws/baz.txt
              permissions: 0644
              contents:
                source:
                  path: assets/etcd/baz.txt
        worker:
          iam:
            policy:
              statements:
              - actions:
                - "ec2:*"
                effect: "Allow"
                resources:
                - "*"
          kubelet:
            nodeLabels:
              role: worker
            featureGates:
              Accelerators: "true"
          systemd:
            units:
            - name: save-queue-name.service
              contents:
                inline: |
                  [Unit]
          storage:
            files:
            - path: /var/kube-aws/bar.txt
              permissions: 0644
              contents:
                inline: worker-bar
            - path: /var/kube-aws/baz.txt
              permissions: 0644
              contents:
                source:
                  path: assets/worker/baz.txt

`,
				},
			},
			assertConfig: []ConfigTester{
				func(c *config.Config, t *testing.T) {
					cp := c.PluginConfigs["myPlugin"]

					if !cp.Enabled {
						t.Errorf("The plugin should have been enabled: %+v", cp)
					}

					if q, ok := cp.Values["queue"].(map[string]interface{}); ok {
						if m, ok := q["name"].(string); ok {
							if m != "baz1" {
								t.Errorf("The plugin should have queue.name set to \"baz1\", but was set to \"%s\"", m)
							}
						}
					}

					np := c.NodePools[0].Plugins["myPlugin"]

					if !np.Enabled {
						t.Errorf("The plugin should have been enabled: %+v", np)
					}

					if q, ok := np.Values["queue"].(map[string]interface{}); ok {
						if m, ok := q["name"].(string); ok {
							if m != "baz2" {
								t.Errorf("The plugin should have queue.name set to \"baz2\", but was set to \"%s\"", m)
							}
						}
					}
				},
			},
			assertCluster: []ClusterTester{
				func(c root.Cluster, t *testing.T) {
					cp := c.ControlPlane()
					np := c.NodePools()[0]
					etcd := c.Etcd()

					{
						e := model.CustomFile{
							Path:        "/var/kube-aws/bar.txt",
							Permissions: 0644,
							Content:     "controller-bar",
						}
						a := cp.StackConfig.Controller.CustomFiles[0]
						if !reflect.DeepEqual(e, a) {
							t.Errorf("Unexpected controller custom file from plugin: expected=%v actual=%v", e, a)
						}
					}
					{
						e := model.CustomFile{
							Path:        "/var/kube-aws/baz.txt",
							Permissions: 0644,
							Content:     "controller-baz",
						}
						a := cp.StackConfig.Controller.CustomFiles[1]
						if !reflect.DeepEqual(e, a) {
							t.Errorf("Unexpected controller custom file from plugin: expected=%v actual=%v", e, a)
						}
					}
					{
						e := model.IAMPolicyStatements{
							model.IAMPolicyStatement{
								Effect:    "Allow",
								Actions:   []string{"ec2:Describe*"},
								Resources: []string{"*"},
							},
						}
						a := cp.StackConfig.Controller.IAMConfig.Policy.Statements
						if !reflect.DeepEqual(e, a) {
							t.Errorf("Unexpected controller iam policy statements from plugin: expected=%v actual=%v", e, a)
						}
					}

					{
						e := model.CustomFile{
							Path:        "/var/kube-aws/bar.txt",
							Permissions: 0644,
							Content:     "etcd-bar",
						}
						a := etcd.StackConfig.Etcd.CustomFiles[0]
						if !reflect.DeepEqual(e, a) {
							t.Errorf("Unexpected etcd custom file from plugin: expected=%v actual=%v", e, a)
						}
					}
					{
						e := model.CustomFile{
							Path:        "/var/kube-aws/baz.txt",
							Permissions: 0644,
							Content:     "etcd-baz",
						}
						a := etcd.StackConfig.Etcd.CustomFiles[1]
						if !reflect.DeepEqual(e, a) {
							t.Errorf("Unexpected etcd custom file from plugin: expected=%v actual=%v", e, a)
						}
					}
					{
						e := model.IAMPolicyStatements{
							model.IAMPolicyStatement{
								Effect:    "Allow",
								Actions:   []string{"ec2:Describe*"},
								Resources: []string{"*"},
							},
						}
						a := etcd.StackConfig.Etcd.IAMConfig.Policy.Statements
						if !reflect.DeepEqual(e, a) {
							t.Errorf("Unexpected etcd iam policy statements from plugin: expected=%v actual=%v", e, a)
						}
					}

					{
						e := model.CustomFile{
							Path:        "/var/kube-aws/bar.txt",
							Permissions: 0644,
							Content:     "worker-bar",
						}
						a := np.StackConfig.CustomFiles[0]
						if !reflect.DeepEqual(e, a) {
							t.Errorf("Unexpected worker custom file from plugin: expected=%v actual=%v", e, a)
						}
					}
					{
						e := model.CustomFile{
							Path:        "/var/kube-aws/baz.txt",
							Permissions: 0644,
							Content:     "worker-baz",
						}
						a := np.StackConfig.CustomFiles[1]
						if !reflect.DeepEqual(e, a) {
							t.Errorf("Unexpected worker custom file from plugin: expected=%v actual=%v", e, a)
						}
					}
					{
						e := model.IAMPolicyStatements{
							model.IAMPolicyStatement{
								Effect:    "Allow",
								Actions:   []string{"ec2:*"},
								Resources: []string{"*"},
							},
						}
						a := np.StackConfig.IAMConfig.Policy.Statements
						if !reflect.DeepEqual(e, a) {
							t.Errorf("Unexpected worker iam policy statements from plugin: expected=%v actual=%v", e, a)
						}
					}

					// A kube-aws plugin can inject systemd units
					controllerUserdataS3Part := cp.UserDataController.Parts[model.USERDATA_S3].Asset.Content
					if !strings.Contains(controllerUserdataS3Part, "save-queue-name.service") {
						t.Errorf("Invalid controller userdata: %v", controllerUserdataS3Part)
					}

					etcdUserdataS3Part := etcd.UserDataEtcd.Parts[model.USERDATA_S3].Asset.Content
					if !strings.Contains(etcdUserdataS3Part, "save-queue-name.service") {
						t.Errorf("Invalid etcd userdata: %v", etcdUserdataS3Part)
					}

					workerUserdataS3Part := np.UserDataWorker.Parts[model.USERDATA_S3].Asset.Content
					if !strings.Contains(workerUserdataS3Part, "save-queue-name.service") {
						t.Errorf("Invalid worker userdata: %v", workerUserdataS3Part)
					}

					// A kube-aws plugin can inject custom cfn stack resources
					controlPlaneStackTemplate, err := cp.RenderStackTemplateAsString()
					if err != nil {
						t.Errorf("failed to render control-plane stack template: %v", err)
					}
					if !strings.Contains(controlPlaneStackTemplate, "QueueFromMyPlugin") {
						t.Errorf("Invalid control-plane stack template: missing resource QueueFromMyPlugin: %v", controlPlaneStackTemplate)
					}
					if !strings.Contains(controlPlaneStackTemplate, `"QueueName":"baz1"`) {
						t.Errorf("Invalid control-plane stack template: missing QueueName baz1: %v", controlPlaneStackTemplate)
					}
					if !strings.Contains(controlPlaneStackTemplate, `"Action":["ec2:Describe*"]`) {
						t.Errorf("Invalid control-plane stack template: missing iam policy statement ec2:Describe*: %v", controlPlaneStackTemplate)
					}

					rootStackTemplate, err := c.RenderStackTemplateAsString()
					if err != nil {
						t.Errorf("failed to render root stack template: %v", err)
					}
					if !strings.Contains(rootStackTemplate, "QueueFromMyPlugin") {
						t.Errorf("Invalid root stack template: missing resource QueueFromMyPlugin: %v", rootStackTemplate)
					}
					if !strings.Contains(rootStackTemplate, `"QueueName":"baz1"`) {
						t.Errorf("Invalid root stack template: missing QueueName baz1: %v", rootStackTemplate)
					}

					nodePoolStackTemplate, err := np.RenderStackTemplateAsString()
					if err != nil {
						t.Errorf("failed to render worker node pool stack template: %v", err)
					}
					if !strings.Contains(nodePoolStackTemplate, "QueueFromMyPlugin") {
						t.Errorf("Invalid worker node pool stack template: missing resource QueueFromMyPlugin: %v", nodePoolStackTemplate)
					}
					if !strings.Contains(nodePoolStackTemplate, `"QueueName":"baz2"`) {
						t.Errorf("Invalid worker node pool stack template: missing QueueName baz2: %v", nodePoolStackTemplate)
					}
					if !strings.Contains(nodePoolStackTemplate, `"QueueName":"baz2"`) {
						t.Errorf("Invalid worker node pool stack template: missing QueueName baz2: %v", nodePoolStackTemplate)
					}
					if !strings.Contains(nodePoolStackTemplate, `"Action":["ec2:*"]`) {
						t.Errorf("Invalid worker node pool stack template: missing iam policy statement ec2:*: %v", nodePoolStackTemplate)
					}

					// A kube-aws plugin can inject node labels
					if !strings.Contains(controllerUserdataS3Part, "role=controller") {
						t.Error("missing controller node label: role=controller")
					}

					if !strings.Contains(workerUserdataS3Part, "role=worker") {
						t.Error("missing worker node label: role=worker")
					}

					// A kube-aws plugin can activate feature gates
					if !strings.Contains(workerUserdataS3Part, `--feature-gates="Accelerators=true"`) {
						t.Error("missing worker feature gate: Accelerators=true")
					}

					// A kube-aws plugin can add volume mounts to apiserver pod
					if !strings.Contains(controllerUserdataS3Part, `mountPath: "/etc/my/creds"`) {
						t.Errorf("missing apiserver volume mount: /etc/my/creds")
					}

					// A kube-aws plugin can add volumes to apiserver pod
					if !strings.Contains(controllerUserdataS3Part, `path: "/etc/my/creds"`) {
						t.Errorf("missing apiserver volume: /etc/my/creds")
					}

					// A kube-aws plugin can add flags to apiserver
					if !strings.Contains(controllerUserdataS3Part, `--oidc-issuer-url=https://login.example.com/`) {
						t.Errorf("missing apiserver flag: --oidc-issuer-url=https://login.example.com/")
					}
				},
			},
		},
	}

	for _, validCase := range validCases {
		t.Run(validCase.context, func(t *testing.T) {
			helper.WithPlugins(validCase.plugins, func() {
				plugins, err := plugin.LoadAll()
				if err != nil {
					t.Errorf("failed to load plugins: %v", err)
					t.FailNow()
				}
				if len(plugins) != len(validCase.plugins) {
					t.Errorf("failed to load plugins: expected %d plugins but loaded %d plugins", len(validCase.plugins), len(plugins))
					t.FailNow()
				}

				configBytes := validCase.clusterYaml
				providedConfig, err := config.ConfigFromBytesWithStubs([]byte(configBytes), plugins, helper.DummyEncryptService{}, helper.DummyCFInterrogator{}, helper.DummyEC2Interrogator{})
				if err != nil {
					t.Errorf("failed to parse config %s: %v", configBytes, err)
					t.FailNow()
				}

				t.Run("AssertConfig", func(t *testing.T) {
					for _, assertion := range validCase.assertConfig {
						assertion(providedConfig, t)
					}
				})

				helper.WithDummyCredentials(func(dummyAssetsDir string) {
					var stackTemplateOptions = root.NewOptions(false, false)
					stackTemplateOptions.AssetsDir = dummyAssetsDir
					stackTemplateOptions.ControllerTmplFile = "../../core/controlplane/config/templates/cloud-config-controller"
					stackTemplateOptions.WorkerTmplFile = "../../core/nodepool/config/templates/cloud-config-worker"
					stackTemplateOptions.EtcdTmplFile = "../../core/etcd/config/templates/cloud-config-etcd"
					stackTemplateOptions.RootStackTemplateTmplFile = "../../core/root/config/templates/stack-template.json"
					stackTemplateOptions.NodePoolStackTemplateTmplFile = "../../core/nodepool/config/templates/stack-template.json"
					stackTemplateOptions.ControlPlaneStackTemplateTmplFile = "../../core/controlplane/config/templates/stack-template.json"
					stackTemplateOptions.EtcdStackTemplateTmplFile = "../../core/etcd/config/templates/stack-template.json"
					stackTemplateOptions.NetworkStackTemplateTmplFile = "../../core/network/config/templates/stack-template.json"

					cluster, err := root.ClusterFromConfig(providedConfig, stackTemplateOptions, false)
					if err != nil {
						t.Errorf("failed to create cluster driver : %v", err)
						t.FailNow()
					}

					t.Run("AssertCluster", func(t *testing.T) {
						for _, assertion := range validCase.assertCluster {
							assertion(cluster, t)
						}
					})

					t.Run("ValidateTemplates", func(t *testing.T) {
						if err := cluster.ValidateTemplates(); err != nil {
							t.Errorf("failed to render stack template: %v", err)
						}
					})

					if os.Getenv("KUBE_AWS_INTEGRATION_TEST") == "" {
						t.Skipf("`export KUBE_AWS_INTEGRATION_TEST=1` is required to run integration tests. Skipping.")
						t.SkipNow()
					} else {
						t.Run("ValidateStack", func(t *testing.T) {
							if !s3URIExists {
								t.Errorf("failed to obtain value for KUBE_AWS_S3_DIR_URI")
								t.FailNow()
							}

							report, err := cluster.ValidateStack()

							if err != nil {
								t.Errorf("failed to validate stack: %s %v", report, err)
							}
						})
					}
				})
			})
		})
	}
}
