package core_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

const cqrsOrderProto = `syntax = "proto3";

package order.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/order";

service OrderService {
  rpc CreateOrder(CreateOrderRequest) returns (Order);
  rpc GetOrder(GetOrderRequest) returns (Order);
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
  rpc UpdateOrder(UpdateOrderRequest) returns (Order);
  rpc DeleteOrder(DeleteOrderRequest) returns (DeleteOrderResponse);
  rpc ArchiveOrder(ArchiveOrderRequest) returns (Order) {
    option (sgo.command) = true;
  }
  rpc FindOrderByEmail(FindOrderByEmailRequest) returns (Order) {
    option (sgo.query) = true;
  }
}

message Order {
  string id = 1;
}
message CreateOrderRequest {
  string name = 1;
}
message GetOrderRequest {
  string id = 1;
}
message ListOrdersRequest {
  int32 page = 1;
}
message ListOrdersResponse {
  repeated Order orders = 1;
}
message UpdateOrderRequest {
  string id = 1;
}
message DeleteOrderRequest {
  string id = 1;
}
message DeleteOrderResponse {
  bool success = 1;
}
message ArchiveOrderRequest {
  string id = 1;
}
message FindOrderByEmailRequest {
  string email = 1;
}
`

var _ = Describe("CQRS classification and generation", func() {
	var (
		root, protoDir, appDir string
		file                   *sgoproto.File
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-core-cqrs-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "order.proto"), []byte(cqrsOrderProto), 0644)).To(Succeed())

		fd, err := sgoproto.Compile(protoDir, "order.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err = sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		p := core.Paths{Module: "demo", Entity: "order"}
		appDir = filepath.Join(root, "internal", "application", "order")

		Expect(core.GenerateCommandsAndQueries(file, fd, p, appDir)).To(Succeed())
	})

	It("classifies Create/Update/Delete as commands and Get/List as queries by naming convention", func() {
		cmdSrc, err := os.ReadFile(filepath.Join(appDir, "command_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(cmdSrc)).To(ContainSubstring("type CreateOrderRequest struct"))
		Expect(string(cmdSrc)).To(ContainSubstring("type UpdateOrderRequest struct"))
		Expect(string(cmdSrc)).To(ContainSubstring("type DeleteOrderRequest struct"))
		Expect(string(cmdSrc)).NotTo(ContainSubstring("GetOrderRequest"))
		Expect(string(cmdSrc)).NotTo(ContainSubstring("ListOrdersRequest"))

		querySrc, err := os.ReadFile(filepath.Join(appDir, "query_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(querySrc)).To(ContainSubstring("type GetOrderRequest struct"))
		Expect(string(querySrc)).To(ContainSubstring("type ListOrdersRequest struct"))
		Expect(string(querySrc)).NotTo(ContainSubstring("CreateOrderRequest"))
	})

	It("honors an explicit (sgo.command)/(sgo.query) override for an RPC the naming convention wouldn't classify", func() {
		cmdSrc, err := os.ReadFile(filepath.Join(appDir, "command_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(cmdSrc)).To(ContainSubstring("type ArchiveOrderRequest struct"))

		querySrc, err := os.ReadFile(filepath.Join(appDir, "query_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(querySrc)).To(ContainSubstring("type FindOrderByEmailRequest struct"))
	})

	It("compiles as real Go code", func() {
		modDir := filepath.Join(root, "gobuild")
		Expect(os.MkdirAll(modDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module cqrstest\n\ngo 1.25.0\n"), 0644)).To(Succeed())

		pkgDir := filepath.Join(modDir, "order")
		Expect(os.MkdirAll(pkgDir, 0755)).To(Succeed())
		for _, name := range []string{"command_gen.go", "query_gen.go"} {
			data, err := os.ReadFile(filepath.Join(appDir, name))
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(filepath.Join(pkgDir, name), data, 0644)).To(Succeed())
		}

		cmd := exec.Command("go", "build", "./...")
		cmd.Dir = modDir
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))
	})
})
