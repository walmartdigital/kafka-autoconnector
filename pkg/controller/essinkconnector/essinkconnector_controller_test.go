package essinkconnector_test

import (
	"context"
	"testing"
	"time"

	"github.com/chinniehendrix/go-kaya/pkg/kafkaconnect"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-cop/operator-utils/pkg/util"
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
		fakeKafkaConnectClient        *mocks.MockKafkaConnectClient
		fakeKafkaConnectClientFactory *mocks.MockKafkaConnectClientFactory
		r                             *essinkconnector.ReconcileESSinkConnector
	)

	BeforeEach(func() {
		fakeEventRecorder = mocks.NewMockEventRecorder(ctrl)
		fakeK8sClient = mocks.NewMockClient(ctrl)
		fakeKafkaConnectClient = mocks.NewMockKafkaConnectClient(ctrl)
		fakeKafkaConnectClientFactory = mocks.NewMockKafkaConnectClientFactory(ctrl)

		config := kafkaconnect.ConnectorConfig{
			Name:                         "amida.logging",
			ConnectorClass:               "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
			DocumentType:                 "log",
			Topics:                       "dumblogger-logs,_ims.logs,_amida.logs,_osiris.logs,_midas.logs,_kimun.logs",
			TopicIndexMap:                "dumblogger-logs:<logs-pd-dumblogger-{now/d}>,_ims.logs:<logs-pd-ims-{now/d}>,_amida.logs:<logs-pd-amida-{now/d}>,_osiris.logs:<logs-pd-osiris-{now/d}>,_midas.logs:<logs-pd-midas-{now/d}>,_kimun.logs:<logs-pd-kimun-{now/d}>",
			BatchSize:                    "100",
			ConnectionURL:                "https://es.tools-flsojt.walmartdigital.cl/kibana",
			KeyIgnore:                    "true",
			SchemaIgnore:                 "true",
			BehaviorOnMalformedDocuments: "ignore",
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
		}
		_ = fakeKafkaConnectClient
	})

	It("should create a new KafkaConnect connector", func() {
		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}

		fakeK8sClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1).SetArg(2, *essink)

		fakeKafkaConnectClientFactory.EXPECT().Create("192.168.64.5:30256", gomock.Any()).Return(
			fakeKafkaConnectClient,
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
		fakeKafkaConnectClient.EXPECT().Delete(essink.Spec.Config.Name).Return(
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

	It("should fail validation because ConnectorConfig is invalid", func() {
		invalidconf := kafkaconnect.ConnectorConfig{
			Name:                         "amida.logging",
			ConnectorClass:               "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
			DocumentType:                 "log",
			Topics:                       "dumblogger-logs,_ims.logs,_amida.logs,_osiris.logs,_midas.logs,_kimun.logs",
			TopicIndexMap:                "dumblogger-logs:<logs-pd-dumblogger-{now/d}>,_ims.logs:<logs-pd-ims-{now/d}>,_amida.logs:<logs-pd-amida-{now/d}>,_osiris.logs:<logs-pd-osiris-{now/d}>,_midas.logs:<logs-pd-midas-{now/d}>,_kimun.logs:<logs-pd-kimun-{now/d}>",
			BatchSize:                    "100",
			ConnectionURL:                "https://es.tools-flsojt.walmartdigital.cl/kibana",
			KeyIgnore:                    "true",
			SchemaIgnore:                 "true",
			BehaviorOnMalformedDocuments: "invalid",
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

		req := reconcile.Request{
			NamespacedName: name,
		}
		_, _ = r.Reconcile(req)
	})

	It("should call the Update function because the ESSinkConnector CR is not initialized yet", func() {
		notinitialized := kafkaconnect.ConnectorConfig{
			// Name is absent, this qualifies as "not initialized"
			ConnectorClass:               "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
			DocumentType:                 "log",
			Topics:                       "dumblogger-logs,_ims.logs,_amida.logs,_osiris.logs,_midas.logs,_kimun.logs",
			TopicIndexMap:                "dumblogger-logs:<logs-pd-dumblogger-{now/d}>,_ims.logs:<logs-pd-ims-{now/d}>,_amida.logs:<logs-pd-amida-{now/d}>,_osiris.logs:<logs-pd-osiris-{now/d}>,_midas.logs:<logs-pd-midas-{now/d}>,_kimun.logs:<logs-pd-kimun-{now/d}>",
			BatchSize:                    "100",
			ConnectionURL:                "https://es.tools-flsojt.walmartdigital.cl/kibana",
			KeyIgnore:                    "true",
			SchemaIgnore:                 "true",
			BehaviorOnMalformedDocuments: "ignore",
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
