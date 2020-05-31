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

// syncReadService 는 mysql 서비스가 없는 경우 서비스를 생성한다
func (r *ReconcileMySQL) syncService(mysql *mysqlv1alpha1.MySQL) error {
	klog.Infof("[%s] syncReadService", mysql.Name)
	// 클러스터로부터 서비스를 가져온다
	mysqlSvc := &corev1.Service{}
	if err := r.client.Get(context.TODO(), getServiceName(mysql), mysqlSvc); err != nil {
		// Not Found 에러가 아닌 경우는 가져오는데 실패한 것이므로 에러를 바로 리턴한다
		if !errors.IsNotFound(err) {
			return err
		}
		// 서비스가 없으므로 생성한다
		klog.Infof("[%s] Could not find mysql service. Create a new one", mysql.Name)
		return r.createService(mysql)
	}
	return nil
}

// getReadServiceName 는 mysql에 대한 서비스 이름과 네임스페이스를 리턴한다
func getServiceName(mysql *mysqlv1alpha1.MySQL) types.NamespacedName {
	return types.NamespacedName{Namespace: mysql.Namespace, Name: mysql.Name}
}

// createReadService 는 새로운 서비스를 생성한다. 이미 서비스가 존재하는 경우 성공한다
func (r *ReconcileMySQL) createService(mysql *mysqlv1alpha1.MySQL) error {
	// 서비스를 위한 객체를 생성한다
	mysqlSvc, err := newService(mysql, r.scheme)
	if err != nil {
		return err
	}
	// 객체를 이용해서 서비스를 생성한다
	if err := r.client.Create(context.TODO(), mysqlSvc); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// newReadService 는 서비스를 위한 객체를 생성한다. 객체는 mysql 객체를 오너로 가진다
func newService(mysql *mysqlv1alpha1.MySQL, scheme *runtime.Scheme) (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      getServiceName(mysql).Name,
			Namespace: getServiceName(mysql).Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "mysql",
					Port: 3306,
				},
			},
			Selector: map[string]string{
				"app": "mysql", // TODO
			},
			ClusterIP: "None",
		},
	}
	if err := controllerutil.SetControllerReference(mysql, svc, scheme); err != nil {
		return nil, err
	}
	return svc, nil
}
