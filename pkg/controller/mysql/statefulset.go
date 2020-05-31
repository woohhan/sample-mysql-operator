package mysql

import (
	"context"
	mysqlv1alpha1 "github.com/woohhan/sample-mysql-operator/pkg/apis/mysql/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// syncStatefulSet 는 mysql 스테이트풀셋이 없는 경우 생성한다
func (r *ReconcileMySQL) syncStatefulSet(mysql *mysqlv1alpha1.MySQL) error {
	klog.Infof("[%s] syncStatefulSet", mysql.Name)
	// 클러스터로부터 스테이트풀셋을 가져온다
	statefulSet := &v1.StatefulSet{}
	if err := r.client.Get(context.TODO(), getStatefulSetName(mysql), statefulSet); err != nil {
		// Not Found 에러가 아닌 경우는 가져오는데 실패한 것이므로 에러를 바로 리턴한다
		if !errors.IsNotFound(err) {
			return err
		}
		// 스테이트풀셋이 없으므로 생성한다
		klog.Infof("[%s] Could not find mysql stateful set. Create a new one", mysql.Name)
		return r.createStatefulSet(mysql)
	}
	return nil
}

// getStatefulSetName 는 mysql 스테이트풀셋에 대한 이름을 리턴한다
func getStatefulSetName(mysql *mysqlv1alpha1.MySQL) types.NamespacedName {
	return types.NamespacedName{Namespace: mysql.Namespace, Name: mysql.Name}
}

// createStatefulSet 는 새로운 스테이트풀셋을 생성한다.
func (r *ReconcileMySQL) createStatefulSet(mysql *mysqlv1alpha1.MySQL) error {
	// 객체를 생성한다
	statefulSet, err := newStatefulSet(mysql, r.scheme)
	if err != nil {
		return err
	}
	// 객체를 이용해서 스테이트풀셋을 생성한다
	if err := r.client.Create(context.TODO(), statefulSet); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// newStatefulSet 는 스테이트풀셋을 위한 객체를 생성한다. 객체는 mysql 객체를 오너로 가진다
// 복잡해 보이지만 이 내용은 관리할 애플리케이션에 실행할 내용이기 때문에 애플리케이션에 따라 달라진다
// 이 내용은 MySQL에서 작업을 수행하기 위한 내용이기 때문에 만약 다른 애플리케이션을 위한 오퍼레이터를 만든다면 그 애플리케이션을 위한 코드가 들어가야 한다
func newStatefulSet(mysql *mysqlv1alpha1.MySQL, scheme *runtime.Scheme) (*v1.StatefulSet, error) {
	replicas := int32(2) // TODO
	statefulSet := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getStatefulSetName(mysql).Name,
			Namespace: getStatefulSetName(mysql).Namespace,
		},
		Spec: v1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "mysql", // TODO
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "mysql",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "conf",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "config-map",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "mysql",
									},
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:  "init-mysql",
							Image: "mysql:5.7",
							Command: []string{
								"bash",
								"-c",
								`set -ex
# Generate mysql server-id from pod ordinal index.
[[ ` + "`" + `hostname` + "`" + ` =~ -([0-9]+)$ ]] || exit 1
ordinal=${BASH_REMATCH[1]}
echo [mysqld] > /mnt/conf.d/server-id.cnf
# Add an offset to avoid reserved server-id=0 value.
echo server-id=$((100 + $ordinal)) >> /mnt/conf.d/server-id.cnf
# Copy appropriate conf.d files from config-map to emptyDir.
if [[ $ordinal -eq 0 ]]; then
  cp /mnt/config-map/master.cnf /mnt/conf.d/
else
  cp /mnt/config-map/slave.cnf /mnt/conf.d/
fi`,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "conf",
									MountPath: "/mnt/conf.d",
								},
								{
									Name:      "config-map",
									MountPath: "/mnt/config-map",
								},
							},
						},
						{
							Name:  "clone-mysql",
							Image: "gcr.io/google-samples/xtrabackup:1.0",
							Command: []string{
								"bash",
								"-c",
								`set -ex
# Skip the clone if data already exists.
[[ -d /var/lib/mysql/mysql ]] && exit 0
# Skip the clone on master (ordinal index 0).
[[ ` + "`" + `hostname` + "`" + ` =~ -([0-9]+)$ ]] || exit 1
ordinal=${BASH_REMATCH[1]}
[[ $ordinal -eq 0 ]] && exit 0
# Clone data from previous peer.
ncat --recv-only mysql-$(($ordinal-1)).mysql 3307 | xbstream -x -C /var/lib/mysql
# Prepare the backup.
xtrabackup --prepare --target-dir=/var/lib/mysql`,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/var/lib/mysql",
									SubPath:   "mysql",
								},
								{
									Name:      "conf",
									MountPath: "/etc/mysql/conf.d",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:5.7",
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_ALLOW_EMPTY_PASSWORD",
									Value: "1",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1Gi")},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/var/lib/mysql",
									SubPath:   "mysql",
								},
								{
									Name:      "conf",
									MountPath: "/etc/mysql/conf.d",
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"mysqladmin", "ping",
										},
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"mysql", "-h", "127.0.0.1", "-e", "SELECT 1",
										},
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       2,
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "mysql",
									ContainerPort: 3306,
								},
							},
						},
						{
							Name:  "xtrabackup",
							Image: "gcr.io/google-samples/xtrabackup:1.0",
							Command: []string{
								"bash", "-c",
								`set -ex
cd /var/lib/mysql

# Determine binlog position of cloned data, if any.
if [[ -f xtrabackup_slave_info && "x$(<xtrabackup_slave_info)" != "x" ]]; then
  # XtraBackup already generated a partial "CHANGE MASTER TO" query
  # because we're cloning from an existing slave. (Need to remove the tailing semicolon!)
  cat xtrabackup_slave_info | sed -E 's/;$//g' > change_master_to.sql.in
  # Ignore xtrabackup_binlog_info in this case (it's useless).
  rm -f xtrabackup_slave_info xtrabackup_binlog_info
elif [[ -f xtrabackup_binlog_info ]]; then
  # We're cloning directly from master. Parse binlog position.
  [[ ` + "`" + `cat xtrabackup_binlog_info` + "`" + ` =~ ^(.*?)[[:space:]]+(.*?)$ ]] || exit 1
  rm -f xtrabackup_binlog_info xtrabackup_slave_info
  echo "CHANGE MASTER TO MASTER_LOG_FILE='${BASH_REMATCH[1]}', MASTER_LOG_POS=${BASH_REMATCH[2]}" > change_master_to.sql.in
fi

# Check if we need to complete a clone by starting replication.
if [[ -f change_master_to.sql.in ]]; then
  echo "Waiting for mysqld to be ready (accepting connections)"
  until mysql -h 127.0.0.1 -e "SELECT 1"; do sleep 1; done

  echo "Initializing replication from clone position"
  mysql -h 127.0.0.1 \
-e "$(<change_master_to.sql.in), \
MASTER_HOST='mysql-0.mysql', \
MASTER_USER='root', \
MASTER_PASSWORD='', \
MASTER_CONNECT_RETRY=10; \
START SLAVE;" || exit 1
  # In case of container restart, attempt this at-most-once.
  mv change_master_to.sql.in change_master_to.sql.orig
fi

# Start a server to send backups when requested by peers.
exec ncat --listen --keep-open --send-only --max-conns=1 3307 -c "xtrabackup --backup --slave-info --stream=xbstream --host=127.0.0.1 --user=root"`,
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "xtrabackup",
									ContainerPort: 3307,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/var/lib/mysql",
									SubPath:   "mysql",
								},
								{
									Name:      "conf",
									MountPath: "/etc/mysql/conf.d",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi")},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							"ReadWriteOnce",
						},
						Resources: corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceStorage: resource.MustParse("2Gi")},
						},
					},
				},
			},
			ServiceName: "mysql", // TODO
		},
	}
	if err := controllerutil.SetControllerReference(mysql, statefulSet, scheme); err != nil {
		return nil, err
	}
	return statefulSet, nil
}
