package mysql

import (
	"context"
	mysqlv1alpha1 "github.com/woohhan/sample-mysql-operator/pkg/apis/mysql/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// syncService() 함수와 대부분이 동일하지만 소스 코드를 읽기 쉽게 하기 위해 분리했다. 자세한 주석은 해당 함수를 참고하길 바란다
func (r *ReconcileMySQL) syncReadService(mysql *mysqlv1alpha1.MySQL) error {
	klog.Infof("[%s] syncReadService", mysql.Name)
	mysqlSvc := &corev1.Service{}
	if err := r.client.Get(context.TODO(), getReadServiceName(mysql), mysqlSvc); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		klog.Infof("[%s] Could not find mysql read service. Create a new one", mysql.Name)
		return r.createReadService(mysql)
	}
	return nil
}

func getReadServiceName(mysql *mysqlv1alpha1.MySQL) types.NamespacedName {
	return types.NamespacedName{Namespace: mysql.Namespace, Name: mysql.Name + "-read"}
}

func (r *ReconcileMySQL) createReadService(mysql *mysqlv1alpha1.MySQL) error {
	mysqlReadService, err := newReadService(mysql, r.scheme)
	if err != nil {
		return err
	}
	if err := r.client.Create(context.TODO(), mysqlReadService); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func newReadService(mysql *mysqlv1alpha1.MySQL, scheme *runtime.Scheme) (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      getReadServiceName(mysql).Name,
			Namespace: getReadServiceName(mysql).Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "mysql",
					Port: 3306,
				},
			},
			Selector: map[string]string{
				"app": "mysql",
			},
		},
	}
	if err := controllerutil.SetControllerReference(mysql, svc, scheme); err != nil {
		return nil, err
	}
	return svc, nil
}
