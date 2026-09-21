package core_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

const orderProto = `syntax = "proto3";

package order.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/order";

message Order {
  option (sgo.aggregate_root) = true;
  string id = 1;
  Money total = 2;
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

message OrderLine {
  string product_id = 1;
  int32 quantity = 2;
}

message CreateOrderRequest {
  string id = 1;
}

message OrderResponse {
  Order order = 1;
}
`

var _ = Describe("GenerateAggregate", func() {
	var (
		root, protoDir, domainDir string
		file                      *sgoproto.File
		fd                        protoreflect.FileDescriptor
		p                         core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-core-aggregate-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "order.proto"), []byte(orderProto), 0644)).To(Succeed())

		fd, err = sgoproto.Compile(protoDir, "order.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err = sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		p = core.Paths{Module: "demo", Entity: "order"}
		domainDir = filepath.Join(root, "internal", "domain", "order")

		Expect(core.GenerateAggregate(file, fd, p, domainDir)).To(Succeed())
	})

	It("generates the aggregate root with PullEvents, its value object, and its domain event", func() {
		content, err := os.ReadFile(filepath.Join(domainDir, "order_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		src := string(content)

		Expect(src).To(ContainSubstring("type Order struct"))
		Expect(src).To(ContainSubstring("events []DomainEvent"))
		Expect(src).To(ContainSubstring("func (a *Order) PullEvents() []DomainEvent"))
		Expect(src).To(ContainSubstring("type Money struct"))
		Expect(src).To(ContainSubstring("type OrderPlacedEvent struct"))
		Expect(src).To(ContainSubstring("func (OrderPlacedEvent) isDomainEvent() {}"))
		Expect(src).To(ContainSubstring("type OrderLine struct"))
	})

	It("excludes Request/Response DTOs from the domain package entirely", func() {
		content, err := os.ReadFile(filepath.Join(domainDir, "order_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		src := string(content)

		Expect(src).NotTo(ContainSubstring("CreateOrderRequest"))
		Expect(src).NotTo(ContainSubstring("OrderResponse"))
	})

	It("creates the owned aggregate.go once, and never overwrites it", func() {
		ownedPath := filepath.Join(domainDir, "aggregate.go")
		content, err := os.ReadFile(ownedPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("Order's business methods"))

		Expect(os.WriteFile(ownedPath, []byte("package order\n\n// hand-written\nfunc (a *Order) MarkPlaced() {\n\ta.events = append(a.events, OrderPlacedEvent{OrderId: a.Id})\n}\n"), 0644)).To(Succeed())

		Expect(core.GenerateAggregate(file, fd, p, domainDir)).To(Succeed())

		after, err := os.ReadFile(ownedPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(after)).To(ContainSubstring("hand-written"))
	})

	It("compiles as real Go code, with a hand-written aggregate method actually using the generated events field", func() {
		ownedPath := filepath.Join(domainDir, "aggregate.go")
		Expect(os.WriteFile(ownedPath, []byte(`package order

func (a *Order) MarkPlaced() {
	a.events = append(a.events, OrderPlacedEvent{OrderId: a.Id})
}
`), 0644)).To(Succeed())

		modDir := filepath.Join(root, "gobuild")
		Expect(os.MkdirAll(modDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module aggregatetest\n\ngo 1.25.0\n"), 0644)).To(Succeed())

		orderPkgDir := filepath.Join(modDir, "order")
		Expect(os.MkdirAll(orderPkgDir, 0755)).To(Succeed())
		for _, name := range []string{"order_gen.go", "aggregate.go"} {
			data, err := os.ReadFile(filepath.Join(domainDir, name))
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(filepath.Join(orderPkgDir, name), data, 0644)).To(Succeed())
		}

		mainSrc := `package main

import "aggregatetest/order"

func main() {
	o := &order.Order{Id: "1"}
	o.MarkPlaced()
	events := o.PullEvents()
	if len(events) != 1 {
		panic("expected 1 event")
	}
}
`
		Expect(os.WriteFile(filepath.Join(modDir, "main.go"), []byte(mainSrc), 0644)).To(Succeed())

		cmd := exec.Command("go", "run", ".")
		cmd.Dir = modDir
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))
	})

	It("generates a repository port and sentinel errors typed directly against the aggregate, no domain import needed", func() {
		Expect(core.GenerateAggregateRepositoryPort(fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateDomainErrors(fd, p, domainDir)).To(Succeed())

		repoSrc, err := os.ReadFile(filepath.Join(domainDir, "repository.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(repoSrc)).To(ContainSubstring("type OrderRepository interface"))
		Expect(string(repoSrc)).To(ContainSubstring("Create(ctx context.Context, a *Order) (*Order, error)"))
		Expect(string(repoSrc)).NotTo(ContainSubstring(`"demo/internal/domain/order"`))

		errSrc, err := os.ReadFile(filepath.Join(domainDir, "errors.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(errSrc)).To(ContainSubstring("var ErrNotFound = errors.New"))

		modDir := filepath.Join(root, "gobuild-repo")
		Expect(os.MkdirAll(modDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module repotest\n\ngo 1.25.0\n"), 0644)).To(Succeed())

		orderPkgDir := filepath.Join(modDir, "order")
		Expect(os.MkdirAll(orderPkgDir, 0755)).To(Succeed())
		for _, name := range []string{"order_gen.go", "aggregate.go", "repository.go", "errors.go"} {
			data, err := os.ReadFile(filepath.Join(domainDir, name))
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(filepath.Join(orderPkgDir, name), data, 0644)).To(Succeed())
		}

		cmd := exec.Command("go", "build", "./...")
		cmd.Dir = modDir
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))
	})
})
