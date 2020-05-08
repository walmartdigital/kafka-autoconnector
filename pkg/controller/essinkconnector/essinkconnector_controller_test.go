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

		ctx    context.Context
		cancel context.CancelFunc
	)

	BeforeEach(func() {
		fakeEventRecorder = mocks.NewMockEventRecorder(ctrl)
		fakeClient = mocks.NewMockClient(ctrl)

		essink = &skynetv1alpha1.ESSinkConnector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "blah",
				Namespace: "default",
			},
			Spec: skynetv1alpha1.ESSinkConnectorSpec{
				Config: kafkaconnect.ConnectorConfig{},
			},
		}

		// objs := []runtime.Object{essink}

		s := scheme.Scheme
		s.AddKnownTypes(skynetv1alpha1.SchemeGroupVersion, essink)

		// cl := fake.NewFakeClient(objs...)

		r = &essinkconnector.ReconcileESSinkConnector{
			ReconcilerBase: util.NewReconcilerBase(
				fakeClient,
				s,
				&rest.Config{},
				fakeEventRecorder,
			),
		}

		ctx, cancel = context.WithCancel(context.Background())
		_ = ctx
		_ = cancel

	})

	It("run Reconcile", func() {
		name := types.NamespacedName{
			Namespace: "default",
			Name:      "blah",
		}
		fakeClient.EXPECT().Get(context.TODO(), name, &skynetv1alpha1.ESSinkConnector{}).Return(
			nil,
		).Times(1)

		req := reconcile.Request{
			NamespacedName: name,
		}
		_, _ = r.Reconcile(req)
	})
})
