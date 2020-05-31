package mysql

import (
	"context"
	mysqlv1alpha1 "github.com/woohhan/sample-mysql-operator/pkg/apis/mysql/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add 는 새로운 MySQL 컨트롤러를 만들고 매니저에 추가합니다.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMySQL{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("mysql-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	// 프라이머리 오브젝트 (MySQL)에 변경이 있으면 조정루프에 진입한다.
	if err := c.Watch(&source.Kind{Type: &mysqlv1alpha1.MySQL{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	// 세컨더리 오브젝트 중 서비스에 변경이 있으면 조정 루프에 진입한다
	if err := c.Watch(&source.Kind{Type: &corev1.Service{}},
		&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &mysqlv1alpha1.MySQL{}}); err != nil {
		return err
	}
	// 세컨더리 오브젝트 중 스테이트풀셋에 변경이 있으면 조정 루프에 진입한다
	if err := c.Watch(&source.Kind{Type: &v1.StatefulSet{}},
		&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &mysqlv1alpha1.MySQL{}}); err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileMySQL implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileMySQL{}

// ReconcileMySQL reconciles a MySQL object
type ReconcileMySQL struct {
	// This client, initialized using mgr.Client() above, is a split client that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile 는 클러스터로부터 MySQL 객체를 읽어와서 MySQL.Spec과 실제 클러스터의 상태를 비교해서 싱크를 맞춘다
func (r *ReconcileMySQL) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Infof("[%s] Start Reconcile for mysql", request.NamespacedName)
	defer func() {
		klog.Infof("[%s] End Reconcile for mysql", request.NamespacedName)
	}()

	// MySQL 인스턴스를 가져온다
	mysql := &mysqlv1alpha1.MySQL{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, mysql); err != nil {
		if errors.IsNotFound(err) {
			// 인스턴스가 없는 것은 인스턴스가 삭제된 직후에 조정루프에 들어온 경우이다.
			// 별다른 처리 없이 바로 리턴한다. 만약 추가적인 삭제 작업이 필요하다면 파이널라이즈를 사용해야 한다.
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// MySQL을 위한 컨피그맵이 존재하는지 확인한다
	if err := r.checkConfigMap(mysql); err != nil {
		return reconcile.Result{}, err
	}

	// MySQL 커스텀 리소스가 관리할 각각의 객체에 대해 조정루프를 실행해서 싱크를 맞춘다
	if err := r.syncService(mysql); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.syncReadService(mysql); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.syncStatefulSet(mysql); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileMySQL) checkConfigMap(mysql *mysqlv1alpha1.MySQL) error {
	config := &corev1.ConfigMap{}
	return r.client.Get(context.TODO(), types.NamespacedName{Namespace: mysql.Namespace, Name: mysql.Name}, config)
}
