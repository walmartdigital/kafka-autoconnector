package essinkconnector_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/walmartdigital/go-kaya/pkg/kafkaconnect"
	skynetv1alpha1 "github.com/walmartdigital/kafka-autoconnector/pkg/apis/skynet/v1alpha1"
	"github.com/walmartdigital/kafka-autoconnector/pkg/controller/essinkconnector"
	"github.com/walmartdigital/kafka-autoconnector/pkg/mocks"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var ctrl *gomock.Controller

func TestAll(t *testing.T) {
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	RegisterFailHandler(Fail)
	RunSpecs(t, "ESSinkConnector Controller tests")
}

var _ = Describe("Run Reconcile", func() {
	const (
		controllerName = "controller_essinkconnector"
	)

	var (
		essink                        *skynetv1alpha1.ESSinkConnector
		fakeEventRecorder             *mocks.MockEventRecorder
		fakeK8sClient                 *mocks.MockClient
		fakeCache                     *mocks.MockCache
		fakeMetrics                   *mocks.MockMetrics
		fakeKafkaConnectClient        *mocks.MockKafkaConnectClient
		fakeKafkaConnectClientFactory *mocks.MockKafkaConnectClientFactory
		r                             *essinkconnector.ReconcileESSinkConnector
	)

	BeforeEach(func() {
		fakeEventRecorder = mocks.NewMockEventRecorder(ctrl)
		fakeK8sClient = mocks.NewMockClient(ctrl)
		fakeKafkaConnectClient = mocks.NewMockKafkaConnectClient(ctrl)
		fakeKafkaConnectClientFactory = mocks.NewMockKafkaConnectClientFactory(ctrl)
		fakeCache = mocks.NewMockCache(ctrl)
		fakeMetrics = mocks.NewMockMetrics(ctrl)

		config := map[string]string{
			"name":                            "amida.logging",
			"connector.class":                 "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
			"type.name":                       "log",
			"topics":                          "dumblogger-logs,_ims.logs,_amida.logs,_osiris.logs,_midas.logs,_kimun.logs",
			"topic.index.map":                 "dumblogger-logs:<logs-pd-dumblogger-{now/d}>,_ims.logs:<logs-pd-ims-{now/d}>,_amida.logs:<logs-pd-amida-{now/d}>,_osiris.logs:<logs-pd-osiris-{now/d}>,_midas.logs:<logs-pd-midas-{now/d}>,_kimun.logs:<logs-pd-kimun-{now/d}>",
			"batch.size":                      "100",
			"connection.url":                  "https://elasticsearch",
			"key.ignore":                      "true",
			"schema.ignore":                   "true",
			"behavior.on.malformed.documents": "ignore",
		}

		essink = &skynetv1alpha1.ESSinkConnector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "amida.logging",
				Namespace: "default",
			},
			Spec: skynetv1alpha1.ESSinkConnectorSpec{
				Config: config,
			},
		}

		controllerutil.AddFinalizer(essink, controllerName)

		s := scheme.Scheme
		s.AddKnownTypes(skynetv1alpha1.SchemeGroupVersion, essink)

		r = &essinkconnector.ReconcileESSinkConnector{
			ReconcilerBase: util.NewReconcilerBase(
				fakeK8sClient,
				s,
				&rest.Config{},
				fakeEventRecorder,
			),
			KafkaConnectClientFactory: fakeKafkaConnectClientFactory,
			Cache:                     fakeCache,
			Metrics:                   fakeMetrics,
		}
	})

	It("should create a new KafkaConnect connector", func() {
		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}

		resp := kafkaconnect.Response{
			Result: "notfound",
		}

		fakeK8sClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1).SetArg(2, *essink)

		fakeKafkaConnectClientFactory.EXPECT().Create("192.168.64.5:30256", gomock.Any()).Return(
			fakeKafkaConnectClient,
			nil,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().Read(essink.Spec.Config["name"]).Return(
			&resp,
			nil,
		).Times(1)

		conObj := kafkaconnect.Connector{
			Name:   essink.Spec.Config["name"],
			Config: essink.Spec.Config,
		}

		resp1 := kafkaconnect.Response{
			Result: "success",
		}

		fakeKafkaConnectClient.EXPECT().Create(conObj).Return(
			&resp1,
			nil,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/total/count").Return(
			0,
			false,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/running/count").Return(
			0,
			false,
		).Times(1)

		fakeK8sClient.EXPECT().Status().Return(
			fakeK8sClient,
		).Times(1)

		fakeK8sClient.EXPECT().Update(context.Background(), gomock.Any()).Return(
			nil,
		).Times(1)

		req := reconcile.Request{
			NamespacedName: name,
		}
		_, _ = r.Reconcile(req)
	})

	It("should update an existing KafkaConnect connector", func() {
		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}

		newconfig := map[string]string{
			"name":                            "amida.logging",
			"connector.class":                 "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
			"type.name":                       "log",
			"topics":                          "dumblogger-logs,_ims.logs,_amida.logs,_osiris.logs,_midas.logs,_kimun.logs",
			"topic.index.map":                 "dumblogger-logs:<logs-pd-dumblogger-{now/d}>,_ims.logs:<logs-pd-ims-{now/d}>,_amida.logs:<logs-pd-amida-{now/d}>,_osiris.logs:<logs-pd-osiris-{now/d}>,_midas.logs:<logs-pd-midas-{now/d}>,_kimun.logs:<logs-pd-kimun-{now/d}>",
			"batch.size":                      "100",
			"connection.url":                  "https://elasticsearch",
			"key.ignore":                      "true",
			"schema.ignore":                   "true",
			"behavior.on.malformed.documents": "warn",
		}

		newConnector := &skynetv1alpha1.ESSinkConnector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "amida.logging",
				Namespace: "default",
			},
			Spec: skynetv1alpha1.ESSinkConnectorSpec{
				Config: newconfig,
			},
		}

		resp := kafkaconnect.Response{
			Result:  "success",
			Payload: essink.Spec.Config,
		}

		controllerutil.AddFinalizer(newConnector, controllerName)

		fakeK8sClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1).SetArg(2, *newConnector)

		fakeKafkaConnectClientFactory.EXPECT().Create("192.168.64.5:30256", gomock.Any()).Return(
			fakeKafkaConnectClient,
			nil,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().Read(essink.Spec.Config["name"]).Return(
			&resp,
			nil,
		).Times(1)

		conObj := kafkaconnect.Connector{
			Name:   newConnector.Spec.Config["name"],
			Config: newConnector.Spec.Config,
		}

		resp1 := kafkaconnect.Response{
			Result: "success",
		}

		fakeKafkaConnectClient.EXPECT().Update(conObj).Return(
			&resp1,
			nil,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/total/count").Return(
			0,
			true,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/running/count").Return(
			0,
			true,
		).Times(1)

		fakeK8sClient.EXPECT().Status().Return(
			fakeK8sClient,
		).Times(1)

		fakeK8sClient.EXPECT().Update(context.Background(), gomock.Any()).Return(
			nil,
		).Times(1)

		req := reconcile.Request{
			NamespacedName: name,
		}
		_, _ = r.Reconcile(req)
	})

	It("should restart a failed KafkaConnect connector and log the event in the cache", func() {
		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}

		task0 := kafkaconnect.Task{
			ID:       0,
			State:    "FAILED",
			WorkerID: "somenode:23444",
		}

		task1 := kafkaconnect.Task{
			ID:       1,
			State:    "FAILED",
			WorkerID: "somenode:23444",
		}

		status := kafkaconnect.Status{
			Name: "blah",
			Connector: kafkaconnect.ConnectorStatus{
				State:    "FAILED",
				WorkerID: "somenode:23444",
			},
			Tasks: []kafkaconnect.Task{
				task0,
				task1,
			},
		}

		statusResp := kafkaconnect.Response{
			Result:  "success",
			Payload: status,
		}

		resp := kafkaconnect.Response{
			Result:  "success",
			Payload: essink.Spec.Config,
		}

		fakeK8sClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1).SetArg(2, *essink)

		fakeKafkaConnectClientFactory.EXPECT().Create("192.168.64.5:30256", gomock.Any()).Return(
			fakeKafkaConnectClient,
			nil,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().Read(essink.Spec.Config["name"]).Return(
			&resp,
			nil,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().GetStatus(essink.Spec.Config["name"]).Return(
			&statusResp,
			nil,
		).Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/total/count", 2).Times(1)
		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/running/count", 0).Times(1)

		fakeMetrics.EXPECT().SetGauge("totalNumTasks", float64(2), "default", "controller_essinkconnector", "amida.logging").Times(1)
		fakeMetrics.EXPECT().SetGauge("numRunningTasks", float64(0), "default", "controller_essinkconnector", "amida.logging").Times(1)

		fakeKafkaConnectClient.EXPECT().RestartConnector(essink.Spec.Config["name"]).Return(
			&statusResp,
			nil,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/restart").Return(
			nil,
			false,
		).Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/restart", 1).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/total/count").Return(
			2,
			true,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/running/count").Return(
			0,
			true,
		).Times(1)

		fakeK8sClient.EXPECT().Status().Return(
			fakeK8sClient,
		).Times(1)

		fakeK8sClient.EXPECT().Update(context.Background(), gomock.Any()).Return(
			nil,
		).Times(1)

		req := reconcile.Request{
			NamespacedName: name,
		}
		_, _ = r.Reconcile(req)
	})

	It("should restart failed tasks and log the event in the cache", func() {
		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}

		task0 := kafkaconnect.Task{
			ID:       0,
			State:    "FAILED",
			WorkerID: "somenode:23444",
		}

		task1 := kafkaconnect.Task{
			ID:       1,
			State:    "FAILED",
			WorkerID: "somenode:23444",
		}

		status := kafkaconnect.Status{
			Name: "blah",
			Connector: kafkaconnect.ConnectorStatus{
				State:    "RUNNING",
				WorkerID: "somenode:23444",
			},
			Tasks: []kafkaconnect.Task{
				task0,
				task1,
			},
		}

		statusResp := kafkaconnect.Response{
			Result:  "success",
			Payload: status,
		}

		resp := kafkaconnect.Response{
			Result:  "success",
			Payload: essink.Spec.Config,
		}

		fakeK8sClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1).SetArg(2, *essink)

		fakeKafkaConnectClientFactory.EXPECT().Create("192.168.64.5:30256", gomock.Any()).Return(
			fakeKafkaConnectClient,
			nil,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().Read(essink.Spec.Config["name"]).Return(
			&resp,
			nil,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().GetStatus(essink.Spec.Config["name"]).Return(
			&statusResp,
			nil,
		).Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/total/count", 2).Times(1)
		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/running/count", 0).Times(1)

		fakeMetrics.EXPECT().SetGauge("totalNumTasks", float64(2), "default", "controller_essinkconnector", "amida.logging").Times(1)
		fakeMetrics.EXPECT().SetGauge("numRunningTasks", float64(0), "default", "controller_essinkconnector", "amida.logging").Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/0/restart").Return(
			nil,
			false,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().RestartTask(essink.Spec.Config["name"], 0).Return(
			&statusResp,
			nil,
		).Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/0/restart", 1).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/1/restart").Return(
			nil,
			false,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().RestartTask(essink.Spec.Config["name"], 1).Return(
			&statusResp,
			nil,
		).Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/1/restart", 1).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/total/count").Return(
			2,
			true,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/running/count").Return(
			0,
			true,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/lasthealthycheck").Return(
			nil,
			false,
		).Times(1)

		fakeMetrics.EXPECT().SetGauge("connectorUptime", float64(0), "default", "controller_essinkconnector", "amida.logging").Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/lasthealthycheck", gomock.Any()).Times(1)

		fakeK8sClient.EXPECT().Status().Return(
			fakeK8sClient,
		).Times(1)

		fakeK8sClient.EXPECT().Update(context.Background(), gomock.Any()).Return(
			nil,
		).Times(1)

		req := reconcile.Request{
			NamespacedName: name,
		}
		_, _ = r.Reconcile(req)
	})

	It("should reach maxTaskRestarts and restart the connector", func() {
		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}

		task0 := kafkaconnect.Task{
			ID:       0,
			State:    "FAILED",
			WorkerID: "somenode:23444",
		}

		task1 := kafkaconnect.Task{
			ID:       1,
			State:    "FAILED",
			WorkerID: "somenode:23444",
		}

		status := kafkaconnect.Status{
			Name: "blah",
			Connector: kafkaconnect.ConnectorStatus{
				State:    "RUNNING",
				WorkerID: "somenode:23444",
			},
			Tasks: []kafkaconnect.Task{
				task0,
				task1,
			},
		}

		statusResp := kafkaconnect.Response{
			Result:  "success",
			Payload: status,
		}

		resp := kafkaconnect.Response{
			Result:  "success",
			Payload: essink.Spec.Config,
		}

		fakeK8sClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1).SetArg(2, *essink)

		fakeKafkaConnectClientFactory.EXPECT().Create("192.168.64.5:30256", gomock.Any()).Return(
			fakeKafkaConnectClient,
			nil,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().Read(essink.Spec.Config["name"]).Return(
			&resp,
			nil,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().GetStatus(essink.Spec.Config["name"]).Return(
			&statusResp,
			nil,
		).Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/total/count", 2).Times(1)
		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/running/count", 0).Times(1)

		fakeMetrics.EXPECT().SetGauge("totalNumTasks", float64(2), "default", "controller_essinkconnector", "amida.logging").Times(1)
		fakeMetrics.EXPECT().SetGauge("numRunningTasks", float64(0), "default", "controller_essinkconnector", "amida.logging").Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/0/restart").Return(
			5,
			true,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/restart").Return(
			nil,
			false,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().RestartConnector(essink.Spec.Config["name"]).Return(
			&resp,
			nil,
		).Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/restart", 1).Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/0/restart", 0).Times(1)
		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/1/restart", 0).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/total/count").Return(
			2,
			true,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/running/count").Return(
			0,
			true,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/lasthealthycheck").Return(
			nil,
			false,
		).Times(1)

		fakeMetrics.EXPECT().SetGauge("connectorUptime", float64(0), "default", "controller_essinkconnector", "amida.logging").Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/lasthealthycheck", gomock.Any()).Times(1)

		fakeK8sClient.EXPECT().Status().Return(
			fakeK8sClient,
		).Times(1)

		fakeK8sClient.EXPECT().Update(context.Background(), gomock.Any()).Return(
			nil,
		).Times(1)

		req := reconcile.Request{
			NamespacedName: name,
		}

		_, _ = r.Reconcile(req)
	})

	It("should reach maxConnectorRestarts and hard reset the connector", func() {
		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}

		task0 := kafkaconnect.Task{
			ID:       0,
			State:    "FAILED",
			WorkerID: "somenode:23444",
		}

		task1 := kafkaconnect.Task{
			ID:       1,
			State:    "FAILED",
			WorkerID: "somenode:23444",
		}

		status := kafkaconnect.Status{
			Name: "blah",
			Connector: kafkaconnect.ConnectorStatus{
				State:    "FAILED",
				WorkerID: "somenode:23444",
			},
			Tasks: []kafkaconnect.Task{
				task0,
				task1,
			},
		}
		conObj := kafkaconnect.Connector{
			Name:   essink.Spec.Config["name"],
			Config: essink.Spec.Config,
		}
		statusResp := kafkaconnect.Response{
			Result:  "success",
			Payload: status,
		}

		resp := kafkaconnect.Response{
			Result:  "success",
			Payload: essink.Spec.Config,
		}

		fakeK8sClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1).SetArg(2, *essink)

		fakeKafkaConnectClientFactory.EXPECT().Create("192.168.64.5:30256", gomock.Any()).Return(
			fakeKafkaConnectClient,
			nil,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().Read(essink.Spec.Config["name"]).Return(
			&resp,
			nil,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().GetStatus(essink.Spec.Config["name"]).Return(
			&statusResp,
			nil,
		).Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/total/count", 2).Times(1)
		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/running/count", 0).Times(1)

		fakeMetrics.EXPECT().SetGauge("totalNumTasks", float64(2), "default", "controller_essinkconnector", "amida.logging").Times(1)
		fakeMetrics.EXPECT().SetGauge("numRunningTasks", float64(0), "default", "controller_essinkconnector", "amida.logging").Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/restart").Return(
			5,
			true,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/hardreset").Return(
			nil,
			false,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().Delete(essink.Spec.Config["name"]).Return(
			&resp,
			nil,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().Create(conObj).Return(
			&resp,
			nil,
		).Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/hardreset", 1).Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/0/restart", 0).Times(1)
		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/tasks/1/restart", 0).Times(1)

		fakeCache.EXPECT().Store("/essinkconnector/connectors/amida.logging/restart", 0).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/total/count").Return(
			2,
			true,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/running/count").Return(
			0,
			true,
		).Times(1)

		fakeK8sClient.EXPECT().Status().Return(
			fakeK8sClient,
		).Times(1)

		fakeK8sClient.EXPECT().Update(context.Background(), gomock.Any()).Return(
			nil,
		).Times(1)

		req := reconcile.Request{
			NamespacedName: name,
		}

		_, _ = r.Reconcile(req)
	})

	It("should delete an existing KafkaConnect connector", func() {
		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}

		ts := metav1.Time{
			Time: time.Now(),
		}

		// Mark object for deletion
		essink.SetDeletionTimestamp(&ts)

		fakeK8sClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1).SetArg(2, *essink)

		fakeKafkaConnectClientFactory.EXPECT().Create("192.168.64.5:30256", gomock.Any()).Return(
			fakeKafkaConnectClient,
			nil,
		).Times(1)

		resp := kafkaconnect.Response{
			Result: "success",
		}
		fakeKafkaConnectClient.EXPECT().Delete(essink.Spec.Config["name"]).Return(
			&resp,
			nil,
		).Times(1).Do(
			func(string) {
				util.RemoveFinalizer(essink, controllerName)
			},
		)

		fakeK8sClient.EXPECT().Update(context.TODO(), essink).Return(
			nil,
		).Times(1)

		req := reconcile.Request{
			NamespacedName: name,
		}
		_, _ = r.Reconcile(req)
	})

	It("should fail validation because ConnectorConfig is invalid (because 'connector.class' field is absent)", func() {
		invalidconf := map[string]string{
			"name": "amida.logging",
			// "connector.class":                 "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
			"type.name":                       "log",
			"topics":                          "dumblogger-logs,_ims.logs,_amida.logs,_osiris.logs,_midas.logs,_kimun.logs",
			"topic.index.map":                 "dumblogger-logs:<logs-pd-dumblogger-{now/d}>,_ims.logs:<logs-pd-ims-{now/d}>,_amida.logs:<logs-pd-amida-{now/d}>,_osiris.logs:<logs-pd-osiris-{now/d}>,_midas.logs:<logs-pd-midas-{now/d}>,_kimun.logs:<logs-pd-kimun-{now/d}>",
			"batch.size":                      "100",
			"connection.url":                  "https://elasticsearch",
			"key.ignore":                      "true",
			"schema.ignore":                   "true",
			"behavior.on.malformed.documents": "warn",
		}

		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}

		conn := &skynetv1alpha1.ESSinkConnector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "amida.logging",
				Namespace: "default",
			},
			Spec: skynetv1alpha1.ESSinkConnectorSpec{
				Config: invalidconf,
			},
		}

		controllerutil.AddFinalizer(conn, controllerName)

		fakeK8sClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1).SetArg(2, *conn)

		fakeEventRecorder.EXPECT().Event(conn, "Warning", "ProcessingError", gomock.Any()).Return().Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/total/count").Return(
			0,
			false,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/running/count").Return(
			0,
			false,
		).Times(1)

		fakeK8sClient.EXPECT().Status().Return(
			fakeK8sClient,
		).Times(1)

		fakeK8sClient.EXPECT().Update(context.Background(), gomock.Any()).Return(
			nil,
		).Times(1)

		req := reconcile.Request{
			NamespacedName: name,
		}
		_, _ = r.Reconcile(req)
	})

	It("should fail because of a KafkaConnect client error when calling the kafkaconnect.Create function", func() {
		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}

		resp := kafkaconnect.Response{
			Result: "notfound",
		}

		fakeK8sClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1).SetArg(2, *essink)

		fakeKafkaConnectClientFactory.EXPECT().Create("192.168.64.5:30256", gomock.Any()).Return(
			fakeKafkaConnectClient,
			nil,
		).Times(1)

		fakeKafkaConnectClient.EXPECT().Read(essink.Spec.Config["name"]).Return(
			&resp,
			nil,
		).Times(1)

		conObj := kafkaconnect.Connector{
			Name:   essink.Spec.Config["name"],
			Config: essink.Spec.Config,
		}

		resp1 := kafkaconnect.Response{
			Result: "error",
		}

		fakeKafkaConnectClient.EXPECT().Create(conObj).Return(
			&resp1,
			errors.New("Cannot connect to Kafka Connect instance"),
		).Times(1)

		fakeEventRecorder.EXPECT().Event(essink, "Warning", "ProcessingError", gomock.Any()).Return().Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/total/count").Return(
			0,
			false,
		).Times(1)

		fakeCache.EXPECT().Load("/essinkconnector/connectors/amida.logging/tasks/running/count").Return(
			0,
			false,
		).Times(1)

		fakeK8sClient.EXPECT().Status().Return(
			fakeK8sClient,
		).Times(1)

		fakeK8sClient.EXPECT().Update(context.Background(), gomock.Any()).Return(
			nil,
		).Times(1)

		req := reconcile.Request{
			NamespacedName: name,
		}
		_, _ = r.Reconcile(req)
	})

	It("should call the Update function because the ESSinkConnector CR is not initialized yet", func() {
		notinitialized := map[string]string{
			// Name is absent, this qualifies as "not initialized"
			"connector.class":                 "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
			"type.name":                       "log",
			"topics":                          "dumblogger-logs,_ims.logs,_amida.logs,_osiris.logs,_midas.logs,_kimun.logs",
			"topic.index.map":                 "dumblogger-logs:<logs-pd-dumblogger-{now/d}>,_ims.logs:<logs-pd-ims-{now/d}>,_amida.logs:<logs-pd-amida-{now/d}>,_osiris.logs:<logs-pd-osiris-{now/d}>,_midas.logs:<logs-pd-midas-{now/d}>,_kimun.logs:<logs-pd-kimun-{now/d}>",
			"batch.size":                      "100",
			"connection.url":                  "https://elasticsearch",
			"key.ignore":                      "true",
			"schema.ignore":                   "true",
			"behavior.on.malformed.documents": "ignore",
		}

		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}

		conn := &skynetv1alpha1.ESSinkConnector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "amida.logging",
				Namespace: "default",
			},
			Spec: skynetv1alpha1.ESSinkConnectorSpec{
				Config: notinitialized,
			},
		}

		fakeK8sClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1).SetArg(2, *conn)

		fakeK8sClient.EXPECT().Update(context.TODO(), gomock.Any()).Return(
			nil,
		).Times(1)

		req := reconcile.Request{
			NamespacedName: name,
		}
		_, _ = r.Reconcile(req)
	})
})
