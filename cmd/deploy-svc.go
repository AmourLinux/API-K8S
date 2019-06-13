package main

import (
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var configPath = filepath.Join(os.Getenv("HOME"), ".kube", "config-cinder")
var kubeConfig = flag.String("kubeconfig", configPath, "Obsolute path")
var result = &appsv1.Deployment{}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	http.HandleFunc("/apis/apps/v1/deployments/", WithBasicAuth(handleDeploy, "dev", "dev"))
	//http.HandlerFunc("/apis/apps/v1/namespaces/{namespace}/deployments", )

	//log.Fatal(http.ListenAndServe(":12380", nil))
	log.Fatal(http.ListenAndServeTLS(":12380", "/Users/oker/.go-pki/api.pem", "/Users/oker/.go-pki/api-key.pem", nil))
}

func handleDeploy(w http.ResponseWriter, r *http.Request) {

	var data = make([]byte, 8192)
	var deploy = &appsv1.Deployment{}

	log.Println("Start handle...")

	log.Printf("Content Length: %v, Path: %v, Host: %v\n", r.ContentLength, r.URL.Path, r.Host)
	//file, err := os.OpenFile("/Users/oker/Desktop/temp", os.O_CREATE|os.O_WRONLY, 0666)
	//if err != nil {
	//	panic(err)
	//}
	//sum, err := io.Copy(file, r.Body)
	//if err != nil {
	//	panic(err)
	//}
	//log.Println("Write: ", sum)

	//strings.Split(r.URL.Path, "/")
	var deployName = strings.TrimPrefix(r.URL.Path, "/apis/apps/v1/deployments/")
	log.Printf("Receive trimed path: %s\n", deployName)

	//if r.Method == "GET" {
	//
	//}

	// Read body
	data, err := ioutil.ReadAll(r.Body)
	//n, err := r.Body.Read(data)
	if err != nil && err != io.EOF {
		//return
		panic(err)
	}
	//log.Println("Received data sum: ", n)
	//data = data[:n]

	log.Printf("Readed data describe: \n%v\n", string(data))

	//err = json.NewDecoder(r.Body).Decode(deploy)
	//if err != nil && err != io.EOF {
	//	panic(err)
	//}
	//p("PKG json, Accept deploy: ", deploy)

	// Json to deploy
	err = json.Unmarshal(data, deploy)
	if err != nil && err != io.EOF {
		log.Println("Error unmarshal deploy: ", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request!"))
		r.Body.Close()
		return
		//panic(err)
	}
	log.Printf("Accept deploy: %v\n", deploy)

	// Initial clientSet
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		panic(err)
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	deploymentClient := clientSet.AppsV1().Deployments("default")
	//deploymentClient := clientSet.AppsV1().Deployments(deploy.Namespace)

	log.Printf("Client method: %s\n", r.Method)

	// change if.. to switch, for indicate client method.
	switch r.Method {
	case "GET":
		result, err = deploymentClient.Get(deploy.Name, metav1.GetOptions{})
		//result, err = deploymentClient.Get(deployName, metav1.GetOptions{})
		//CheckError(err, w, r)
		if err != nil {
			log.Println(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			r.Body.Close()
			return
		}
		log.Println("Success get deploy", result)
	case "POST":
		result, err = deploymentClient.Create(deploy)
		if err != nil {
			log.Println(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			r.Body.Close()
			return
		}
		log.Println("Success create deploy: ", result)
	case "PUT":
		//ns := deploy.Namespace
		name := deploy.Name
		result, err = deploymentClient.Get(name, metav1.GetOptions{})
		if err != nil {
			result, err = deploymentClient.Create(deploy)
			if err != nil {
				log.Println("Error:", err.Error())
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				r.Body.Close()
				return
			}
			log.Println("Success create deploy: ", result)
		} else {
			result, err = deploymentClient.Update(deploy)
			if err != nil {
				log.Println("Error:", err.Error())
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(err.Error()))
				r.Body.Close()
				return
			}
			log.Println("Success update deploy: ", result)
		}
	case "PATCH":
		result, err = deploymentClient.Patch(deployName, types.PatchType(r.Header.Get("Content-Type")), data)
		//result, err = deploymentClient.Patch(deployName, types.StrategicMergePatchType, data)
		if err != nil {
			log.Println("Error:", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			r.Body.Close()
			return
		}
		log.Println("Success patch deploy: ", result)
	case "DELETE":
		err = deploymentClient.Delete(deploy.Name, &metav1.DeleteOptions{})
		if err != nil {
			log.Println("Error:", err.Error())
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(err.Error()))
			r.Body.Close()
			return
		}
		log.Println("Success delete deploy: ", deploy.Name)
	default:
		log.Println("Not found method: ", r.Method)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Not found method: " + r.Method))
		r.Body.Close()
		return
	}

	//w.WriteHeader(http.StatusCreated)
	w.Header().Add("Content-type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	resp, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	_, err = w.Write(resp)
	if err != nil {
		panic(err)
	}

	//v1.DeploymentInterface()
	//deploymentClient.Update()
	//deploymentClient.UpdateScale()
}

func createClient(kubeconfig *string) {

}

func createDeploy(deploymentClient v1.DeploymentInterface, deploy *appsv1.Deployment) (*appsv1.Deployment, error) {
	result, err := deploymentClient.Create(deploy)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func tempPatch() {
	/*
			newDeploy := `{
		    "kind": "Deployment",
		    "apiVersion": "apps/v1",
		    "metadata": {
		        "name": "nginx",
		        "namespace": "kube-system",
		        "labels": {
		            "run": "nginx"
		        }
		    },
		    "spec": {
		        "replicas": 1,
		        "selector": {
		            "matchLabels": {
		                "run": "nginx"
		            }
		        },
		        "template": {
		            "metadata": {
		                "labels": {
		                    "run": "nginx"
		                }
		            },
		            "spec": {
		                "containers": [
							{
		                        "name": "nginx",
		                        "image": "nginx:stable",
		                        "imagePullPolicy": "IfNotPresent",
		                    }
		                ],
		                "restartPolicy": "Always"
		            }
		        }
		    }
		}`
	*/

	// todo: try used patch API
	//strategicpatch.CreateTwoWayMergePatch()
}

func WithBasicAuth(handlerFunc http.HandlerFunc, username, password string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// check password
		log.Println("Check password.")
		user, pass, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized!"))
			r.Body.Close()
			return
		}

		// handle request
		handlerFunc(w, r)
	}
}

func WithHttpsCheck(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// check https
		log.Println("Check HTTPS.")
		if r.Proto != "" {
			fmt.Println(r.Proto, r.Host, r.Method, r.Header, r.RemoteAddr, r.RequestURI, r.URL, r.UserAgent())
			w.WriteHeader(http.StatusPermanentRedirect)
			r.Body.Close()
			return
		}

		// handle request
		handlerFunc(w, r)
	}
}

//func CheckDeployExist(name string, options metav1.GetOptions) bool {
//
//}

func CheckError(err error, w http.ResponseWriter, r *http.Request) {
	if err != nil {
		log.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		r.Body.Close()
		return
	}
}
