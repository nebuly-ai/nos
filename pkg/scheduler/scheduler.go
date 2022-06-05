package scheduler

import (
	v1 "k8s.io/api/core/v1"
)

/*
import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/apimachinery/pkg/util/wait"
	"errors"
)
*/

const schedulerName = "neb-scheduler"

type Scheduler struct {
	podQueue chan *v1.Pod
}
