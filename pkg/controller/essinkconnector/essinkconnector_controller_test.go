package essinkconnector_test

import (
	"context"
	"testing"

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
	var (
		essink            *skynetv1alpha1.ESSinkConnector
		fakeEventRecorder *mocks.MockEventRecorder
		fakeClient        *mocks.MockClient
		r                 *essinkconnector.ReconcileESSinkConnector
	)

	BeforeEach(func() {
		fakeEventRecorder = mocks.NewMockEventRecorder(ctrl)
		fakeClient = mocks.NewMockClient(ctrl)

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
				Name:      "blah",
				Namespace: "default",
			},
			Spec: skynetv1alpha1.ESSinkConnectorSpec{
				Config: config,
			},
		}

		s := scheme.Scheme
		s.AddKnownTypes(skynetv1alpha1.SchemeGroupVersion, essink)

		r = &essinkconnector.ReconcileESSinkConnector{
			ReconcilerBase: util.NewReconcilerBase(
				fakeClient,
				s,
				&rest.Config{},
				fakeEventRecorder,
			),
		}
	})

	It("run Reconcile", func() {
		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}
		fakeClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1).SetArg(2, *essink)

		req := reconcile.Request{
			NamespacedName: name,
		}
		_, _ = r.Reconcile(req)
	})
})
