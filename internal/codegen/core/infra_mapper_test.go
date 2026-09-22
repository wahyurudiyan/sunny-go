package core_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/wiregen"
)

const mapperOrderProto = `syntax = "proto3";

package order.v1;

import "sgo/options.proto";
import "buf/validate/validate.proto";

option go_package = "demo/contract/gen/order";

service OrderService {
  rpc CreateOrder(CreateOrderRequest) returns (Order);
}

message Order {
  option (sgo.aggregate_root) = true;
  string id = 1;
  string email = 2 [(buf.validate.field).string.email = true];
  Money total = 3;
}

message Money {
  option (sgo.value_object) = true;
  int64 amount = 1 [(buf.validate.field).int64.gt = 0];
  string currency = 2;
}

message CreateOrderRequest {
  string email = 1 [(buf.validate.field).string.email = true];
}
`

var _ = Describe("GenerateInfraMapper", func() {
	var (
		root, protoDir, domainDir, appDir, mapperDir, eventDir, wireDir string
		file                                                            *sgoproto.File
		p                                                               core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-core-infra-mapper-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "order.proto"), []byte(mapperOrderProto), 0644)).To(Succeed())

		fd, err := sgoproto.Compile(protoDir, "order.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err = sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		p = core.Paths{Module: "demo", Entity: "order"}
		domainDir = filepath.Join(root, "internal", "domain", "order")
		appDir = filepath.Join(root, "internal", "application", "order")
		mapperDir = filepath.Join(root, "internal", "infrastructure", "transport")
		eventDir = filepath.Join(root, "internal", "domain", "event")
		wireDir = filepath.Join(root, "contract", "gen", "order")

		Expect(core.GenerateEventKernel(eventDir)).To(Succeed())
		Expect(core.GenerateAggregate(file, fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateCommandsAndQueries(file, fd, p, appDir)).To(Succeed())
		Expect(wiregen.Generate(fd, wireDir)).To(Succeed())
		Expect(core.GenerateInfraMapper(file, p, mapperDir)).To(Succeed())
	})

	It("generates ToDomain/FromDomain for the aggregate and its value object, excluding nothing else", func() {
		src, err := os.ReadFile(filepath.Join(mapperDir, "order_mapper_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		s := string(src)

		Expect(s).To(ContainSubstring("func OrderToDomain(w *wire.Order) (*domain.Order, error)"))
		Expect(s).To(ContainSubstring("func OrderFromDomain(d *domain.Order) *wire.Order"))
		Expect(s).To(ContainSubstring("func MoneyToDomain(w *wire.Money) (*domain.Money, error)"))
		Expect(s).To(ContainSubstring("validator.Validate(w)"))
	})

	It("generates a ToApp conversion for each Request DTO, for the gRPC adapter's own request-side mapping", func() {
		src, err := os.ReadFile(filepath.Join(mapperDir, "order_mapper_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		s := string(src)

		Expect(s).To(ContainSubstring("func CreateOrderRequestToApp(w *wire.CreateOrderRequest) (*app.CreateOrderRequest, error)"))
		Expect(s).NotTo(ContainSubstring("OrderToApp"), "the aggregate itself isn't a DTO and shouldn't get a ToApp conversion")
	})

	It("compiles against the real protoc-gen-go output and rejects/accepts data per the proto's real constraints", func() {
		modDir := filepath.Join(root, "gobuild")
		Expect(os.MkdirAll(modDir, 0755)).To(Succeed())

		// The generated wire code blank-imports pkg/sgoproto (to register
		// sgo/options.proto's extensions), a real, publicly resolvable
		// package of this same module once pushed — `go mod tidy` in a
		// real generated project resolves it from the module proxy like
		// any other dependency, no replace needed. Here, this repo's own
		// working tree is ahead of what's pushed, so the replace points
		// `go mod tidy` at the local checkout instead of the stale remote,
		// purely a test-harness accommodation for in-flight development.
		repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
		Expect(err).NotTo(HaveOccurred())
		goMod := "module demo\n\ngo 1.26.0\n\nrequire github.com/wahyurudiyan/sunny-go v0.0.0-00010101000000-000000000000\n\nreplace github.com/wahyurudiyan/sunny-go => " + repoRoot + "\n"
		Expect(os.WriteFile(filepath.Join(modDir, "go.mod"), []byte(goMod), 0644)).To(Succeed())

		copyDir := func(src, dstRel string) {
			GinkgoHelper()
			entries, err := os.ReadDir(src)
			Expect(err).NotTo(HaveOccurred())
			dst := filepath.Join(modDir, dstRel)
			Expect(os.MkdirAll(dst, 0755)).To(Succeed())
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				data, err := os.ReadFile(filepath.Join(src, e.Name()))
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(dst, e.Name()), data, 0644)).To(Succeed())
			}
		}

		copyDir(eventDir, "internal/domain/event")
		copyDir(domainDir, "internal/domain/order")
		copyDir(appDir, "internal/application/order")
		copyDir(mapperDir, "internal/infrastructure/transport")
		copyDir(wireDir, "contract/gen/order")

		mainSrc := `package main

import (
	"fmt"

	mapper "demo/internal/infrastructure/transport"
	wire "demo/contract/gen/order"
)

func main() {
	bad := &wire.Order{Id: "1", Email: "not-an-email", Total: &wire.Money{Amount: -5, Currency: "USD"}}
	_, err := mapper.OrderToDomain(bad)
	if err == nil {
		panic("expected validation error for bad order")
	}
	fmt.Println("rejected bad order:", err)

	good := &wire.Order{Id: "1", Email: "user@example.com", Total: &wire.Money{Amount: 100, Currency: "USD"}}
	domainOrder, err := mapper.OrderToDomain(good)
	if err != nil {
		panic("expected no error for good order: " + err.Error())
	}
	if domainOrder.Id != "1" || domainOrder.Total.Amount != 100 {
		panic("mapped fields don't match")
	}

	backToWire := mapper.OrderFromDomain(domainOrder)
	if backToWire.Email != "user@example.com" {
		panic("round-trip mismatch")
	}
	fmt.Println("accepted good order, round-tripped ok")

	_, err = mapper.CreateOrderRequestToApp(&wire.CreateOrderRequest{Email: "not-an-email"})
	if err == nil {
		panic("expected validation error for bad CreateOrderRequest")
	}
	fmt.Println("rejected bad CreateOrderRequest:", err)

	appReq, err := mapper.CreateOrderRequestToApp(&wire.CreateOrderRequest{Email: "user@example.com"})
	if err != nil {
		panic("expected no error for good CreateOrderRequest: " + err.Error())
	}
	if appReq.Email != "user@example.com" {
		panic("CreateOrderRequestToApp field mismatch")
	}
	fmt.Println("accepted good CreateOrderRequest")
}
`
		Expect(os.WriteFile(filepath.Join(modDir, "main.go"), []byte(mainSrc), 0644)).To(Succeed())

		tidy := exec.Command("go", "mod", "tidy")
		tidy.Dir = modDir
		out, err := tidy.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))

		cmd := exec.Command("go", "run", ".")
		cmd.Dir = modDir
		out, err = cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))
		Expect(string(out)).To(ContainSubstring("rejected bad order"))
		Expect(string(out)).To(ContainSubstring("accepted good order, round-tripped ok"))
		Expect(string(out)).To(ContainSubstring("rejected bad CreateOrderRequest"))
		Expect(string(out)).To(ContainSubstring("accepted good CreateOrderRequest"))
	})
})
