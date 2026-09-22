package proto_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

var _ = Describe("vendored proto options", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-proto-wellknown-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	})

	compile := func(content string) protoreflect.FileDescriptor {
		GinkgoHelper()
		path := filepath.Join(dir, "order.proto")
		Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())

		fd, err := proto.Compile(dir, "order.proto")
		Expect(err).NotTo(HaveOccurred())
		return fd
	}

	Context("sgo's own custom options (sgo/options.proto)", func() {
		It("compiles a proto importing sgo/options.proto and reports marked roles/CQRS classification", func() {
			content := `syntax = "proto3";

package order.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/order";

message Order {
  option (sgo.aggregate_root) = true;
  string id = 1;
}

message Money {
  option (sgo.value_object) = true;
  int64 amount = 1;
  string currency = 2;
}

message OrderPlacedEvent {
  option (sgo.domain_event) = true;
  string order_id = 1;
}

message PlainMessage {
  string note = 1;
}

service OrderService {
  rpc PlaceOrder(Order) returns (Order) {
    option (sgo.command) = true;
  }
  rpc FindOrderByEmail(Order) returns (Order) {
    option (sgo.query) = true;
  }
  rpc UnmarkedMethod(Order) returns (Order);
}
`
			fd := compile(content)
			msgs := fd.Messages()

			Expect(proto.IsAggregateRoot(msgs.ByName("Order"))).To(BeTrue())
			Expect(proto.IsValueObject(msgs.ByName("Money"))).To(BeTrue())
			Expect(proto.IsDomainEvent(msgs.ByName("OrderPlacedEvent"))).To(BeTrue())

			// negative cases: a message marked one way isn't also reported
			// as the others, and a wholly unmarked message is none of them.
			Expect(proto.IsValueObject(msgs.ByName("Order"))).To(BeFalse())
			Expect(proto.IsDomainEvent(msgs.ByName("Order"))).To(BeFalse())
			Expect(proto.IsAggregateRoot(msgs.ByName("PlainMessage"))).To(BeFalse())
			Expect(proto.IsValueObject(msgs.ByName("PlainMessage"))).To(BeFalse())
			Expect(proto.IsDomainEvent(msgs.ByName("PlainMessage"))).To(BeFalse())

			svc := fd.Services().Get(0)
			methods := svc.Methods()

			Expect(proto.IsCommand(methods.ByName("PlaceOrder"))).To(BeTrue())
			Expect(proto.IsQuery(methods.ByName("PlaceOrder"))).To(BeFalse())
			Expect(proto.IsQuery(methods.ByName("FindOrderByEmail"))).To(BeTrue())
			Expect(proto.IsCommand(methods.ByName("FindOrderByEmail"))).To(BeFalse())
			Expect(proto.IsCommand(methods.ByName("UnmarkedMethod"))).To(BeFalse())
			Expect(proto.IsQuery(methods.ByName("UnmarkedMethod"))).To(BeFalse())
		})
	})

	Context("vendored protovalidate (buf/validate/validate.proto)", func() {
		It("compiles a proto importing buf/validate/validate.proto and reports real field constraints", func() {
			content := `syntax = "proto3";

package order.v1;

import "buf/validate/validate.proto";

option go_package = "demo/contract/gen/order";

message Order {
  string email = 1 [(buf.validate.field).string.email = true];
  int32 quantity = 2 [(buf.validate.field).int32.gt = 0];
  string unconstrained = 3;
}
`
			fd := compile(content)
			md := fd.Messages().ByName("Order")

			emailRules := proto.FieldConstraints(md.Fields().ByName("email"))
			Expect(emailRules).NotTo(BeNil())
			Expect(emailRules.GetString_().GetEmail()).To(BeTrue())

			quantityRules := proto.FieldConstraints(md.Fields().ByName("quantity"))
			Expect(quantityRules).NotTo(BeNil())
			Expect(quantityRules.GetInt32().GetGt()).To(Equal(int32(0)))

			Expect(proto.FieldConstraints(md.Fields().ByName("unconstrained"))).To(BeNil())
		})
	})

	Context("vendored google.api.http (google/api/annotations.proto) and sgo.base_path", func() {
		It("compiles a proto importing google/api/annotations.proto and reports the real HttpRule", func() {
			content := `syntax = "proto3";

package order.v1;

import "google/api/annotations.proto";
import "sgo/options.proto";

option go_package = "demo/contract/gen/order";

service OrderService {
  option (sgo.base_path) = "/v2";

  rpc GetOrder(GetOrderRequest) returns (Order) {
    option (google.api.http) = {
      get: "/orders/{order_id}"
    };
  }
  rpc PlainRPC(GetOrderRequest) returns (Order);
}

message GetOrderRequest {
  string order_id = 1;
}

message Order {
  string id = 1;
}
`
			fd := compile(content)
			svc := fd.Services().Get(0)

			rule := proto.HTTPRule(svc.Methods().ByName("GetOrder"))
			Expect(rule).NotTo(BeNil())
			Expect(rule.GetGet()).To(Equal("/orders/{order_id}"))

			Expect(proto.HTTPRule(svc.Methods().ByName("PlainRPC"))).To(BeNil())

			Expect(proto.BasePath(svc)).To(Equal("/v2"))
		})

		It("reports an empty base_path when the service doesn't set one", func() {
			content := `syntax = "proto3";

package order.v1;

option go_package = "demo/contract/gen/order";

service OrderService {
  rpc GetOrder(GetOrderRequest) returns (Order);
}

message GetOrderRequest {
  string order_id = 1;
}

message Order {
  string id = 1;
}
`
			fd := compile(content)
			Expect(proto.BasePath(fd.Services().Get(0))).To(Equal(""))
		})
	})
})
