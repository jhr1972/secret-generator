package main

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Cluster struct {
	Name        string `yaml:"name"`
	SaName      string `yaml:"sa-name"`
	SaNamespace string `yaml:"sa-namespace"`
	ArgoNs      string `yaml:"argo-ns"`
	ArgoRef     string `yaml:"argo-ref"`
	UrlRef      string `yaml:"url-ref"`
	ArgoRefB64  string
	UrlRefB64   string
	Token       []byte
	//	Config      configJSON
	ConfigB64 string
}

//type configJSON struct {
//	bearerToken string `json:"token"`
//}

type Leafclusters struct {
	LeafCluster []Cluster `leafclusters`
}

func main() {

	var mode string
	var pathprefix string
	flag.StringVar(&mode, "m", "k8s", "Specify run mode \"native\". Default is k8s mode")

	flag.Usage = func() {
		fmt.Printf("Usage of program: \n")
		fmt.Printf(os.Args[0] + " -m \n")
		// flag.PrintDefaults()  // prints default usage
	}
	flag.Parse()
	fmt.Printf("Running in mode %s \n", mode)
	// Create a traced mux router
	var kubeconfig *string
	for {
		if mode == "native" {

			if home := homedir.HomeDir(); home != "" {
				kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
			} else {
				kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
			}
			pathprefix = ""
			break
		}
		if mode == "k8s" {
			fmt.Printf("Running in k8s mode")
			pathprefix = "/mnt/"
			break

		}

		fmt.Printf("Invalid run mode. Needs to be native or k8s.")
		os.Exit(-1)

	}
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	f, err := os.ReadFile(pathprefix + "config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	out, err := os.Create("newsecret.yaml")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			panic(err)
		}
	}()

	var lcs Leafclusters
	if err := yaml.Unmarshal(f, &lcs); err != nil {
		log.Fatal(err)
	}
	var tmplFile = pathprefix + "secret.j2"
	for i, v := range lcs.LeafCluster {
		fmt.Printf("Name: %s\n", string(v.Name))
		secret, err := clientset.CoreV1().Secrets(string(v.SaNamespace)).Get(context.TODO(), string(v.SaName), metav1.GetOptions{})
		lcs.LeafCluster[i].Token = secret.Data["token"]
		if len(lcs.LeafCluster[i].Token) == 0 {
			fmt.Printf("Secret %s not found \n", v.SaName)
			panic(fmt.Sprintf("Secret %s not found ", v.SaName))

		}
		lcs.LeafCluster[i].ArgoRefB64 = b64.StdEncoding.EncodeToString([]byte(lcs.LeafCluster[i].ArgoRef))
		lcs.LeafCluster[i].UrlRefB64 = b64.StdEncoding.EncodeToString([]byte(lcs.LeafCluster[i].UrlRef))

		type Config struct {
			BearerToken     string          `json:"bearerToken"`
			TlsClientConfig json.RawMessage `json:"tlsClientConfig"`
		}

		data := &Config{
			BearerToken:     string(secret.Data["token"]),
			TlsClientConfig: json.RawMessage(`{ "insecure": true }`),
		}

		b, err := json.Marshal(data)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Config\n%s", string(b))
		lcs.LeafCluster[i].ConfigB64 = b64.StdEncoding.EncodeToString(b)

		tmpl, err := template.New(tmplFile).ParseFiles(tmplFile)
		if err != nil {
			panic(err)
		}
		//err = tmpl.Execute(os.Stdout, lcs.LeafCluster[i])
		err = tmpl.Execute(out, lcs.LeafCluster[i])
		if err != nil {
			panic(err)
		}

	}

}
