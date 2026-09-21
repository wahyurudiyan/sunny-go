package core_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

var _ = Describe("role inference", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-core-roles-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	})

	compile := func(content string) protoreflect.FileDescriptor {
		GinkgoHelper()
		path := filepath.Join(dir, "order.proto")
		Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())
		fd, err := sgoproto.Compile(dir, "order.proto")
		Expect(err).NotTo(HaveOccurred())
		return fd
	}

	Describe("AggregateRoot", func() {
		It("returns the message explicitly marked (sgo.aggregate_root)", func() {
			fd := compile(`syntax = "proto3";
package order.v1;
import "sgo/options.proto";
option go_package = "demo/contract/gen/order";

message Order {
  option (sgo.aggregate_root) = true;
  string id = 1;
}
message SomethingElse {
  string id = 1;
}
`)
			md, err := core.AggregateRoot(fd, "order")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(md.Name())).To(Equal("Order"))
		})

		It("falls back to the message matching the entity name when nothing's marked", func() {
			fd := compile(`syntax = "proto3";
package order.v1;
option go_package = "demo/contract/gen/order";

message Order {
  string id = 1;
}
message CreateOrderRequest {
  string id = 1;
}
`)
			md, err := core.AggregateRoot(fd, "order")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(md.Name())).To(Equal("Order"))
		})

		It("errors when nothing's marked and no message matches the entity name", func() {
			fd := compile(`syntax = "proto3";
package order.v1;
option go_package = "demo/contract/gen/order";

message SomethingElse {
  string id = 1;
}
`)
			_, err := core.AggregateRoot(fd, "order")
			Expect(err).To(MatchError(ContainSubstring("no aggregate root found")))
		})

		It("errors when more than one message is marked", func() {
			fd := compile(`syntax = "proto3";
package order.v1;
import "sgo/options.proto";
option go_package = "demo/contract/gen/order";

message Order {
  option (sgo.aggregate_root) = true;
  string id = 1;
}
message Payment {
  option (sgo.aggregate_root) = true;
  string id = 1;
}
`)
			_, err := core.AggregateRoot(fd, "order")
			Expect(err).To(MatchError(ContainSubstring("more than one message marked")))
		})
	})

	Describe("ValueObjects and DomainEvents", func() {
		It("returns exactly the messages marked, in file order, and nothing else", func() {
			fd := compile(`syntax = "proto3";
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
}
message Address {
  option (sgo.value_object) = true;
  string city = 1;
}
message OrderPlacedEvent {
  option (sgo.domain_event) = true;
  string order_id = 1;
}
message PlainMessage {
  string note = 1;
}
`)
			vos := core.ValueObjects(fd)
			names := make([]string, len(vos))
			for i, md := range vos {
				names[i] = string(md.Name())
			}
			Expect(names).To(Equal([]string{"Money", "Address"}))

			events := core.DomainEvents(fd)
			Expect(events).To(HaveLen(1))
			Expect(string(events[0].Name())).To(Equal("OrderPlacedEvent"))
		})

		It("returns nothing when the proto doesn't import sgo/options.proto at all", func() {
			fd := compile(`syntax = "proto3";
package order.v1;
option go_package = "demo/contract/gen/order";

message Order {
  string id = 1;
}
`)
			Expect(core.ValueObjects(fd)).To(BeEmpty())
			Expect(core.DomainEvents(fd)).To(BeEmpty())
		})
	})

	Describe("IsDTO", func() {
		It("recognizes the Request/Response naming convention", func() {
			Expect(core.IsDTO("CreateOrderRequest")).To(BeTrue())
			Expect(core.IsDTO("OrderResponse")).To(BeTrue())
			Expect(core.IsDTO("Order")).To(BeFalse())
			Expect(core.IsDTO("Money")).To(BeFalse())
		})
	})
})
