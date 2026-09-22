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

const appOrderProto = `syntax = "proto3";

package order.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/order";

service OrderService {
  rpc CreateOrder(CreateOrderRequest) returns (Order);
  rpc GetOrder(GetOrderRequest) returns (Order);
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
  rpc DeleteOrder(DeleteOrderRequest) returns (DeleteOrderResponse);
}

message Order {
  option (sgo.aggregate_root) = true;
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
message DeleteOrderRequest {
  string id = 1;
}
message DeleteOrderResponse {
  bool success = 1;
}
`

var _ = Describe("GenerateApplicationService and GenerateEventPublisher", func() {
	var (
		root, protoDir, domainDir, appDir, portsDir, eventDir string
		file                                                  *sgoproto.File
		p                                                     core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-core-application-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "order.proto"), []byte(appOrderProto), 0644)).To(Succeed())

		fd, err := sgoproto.Compile(protoDir, "order.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err = sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		p = core.Paths{Module: "demo", Entity: "order"}
		domainDir = filepath.Join(root, "internal", "domain", "order")
		appDir = filepath.Join(root, "internal", "application", "order")
		portsDir = filepath.Join(root, "internal", "application", "ports")
		eventDir = filepath.Join(root, "internal", "domain", "event")

		Expect(core.GenerateEventKernel(eventDir)).To(Succeed())
		Expect(core.GenerateAggregate(file, fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateAggregateRepositoryPort(fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateDomainErrors(fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateCommandsAndQueries(file, fd, p, appDir)).To(Succeed())
		Expect(core.GenerateEventPublisher(p, portsDir)).To(Succeed())
		Expect(core.GenerateApplicationService(file, fd, p, appDir)).To(Succeed())
	})

	It("generates a service depending on the aggregate repository and EventPublisher, with return shape per RPC naming convention", func() {
		src, err := os.ReadFile(filepath.Join(appDir, "service.go"))
		Expect(err).NotTo(HaveOccurred())
		s := string(src)

		Expect(s).To(ContainSubstring("type OrderService struct"))
		Expect(s).To(ContainSubstring("repo      domain.OrderRepository"))
		Expect(s).To(ContainSubstring("publisher ports.EventPublisher"))
		Expect(s).To(ContainSubstring("func NewOrderService(repo domain.OrderRepository, publisher ports.EventPublisher) *OrderService"))

		Expect(s).To(ContainSubstring("func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*domain.Order, error)"))
		Expect(s).To(ContainSubstring("func (s *OrderService) GetOrder(ctx context.Context, req *GetOrderRequest) (*domain.Order, error)"))
		Expect(s).To(ContainSubstring("func (s *OrderService) ListOrders(ctx context.Context, req *ListOrdersRequest) ([]*domain.Order, error)"))
		Expect(s).To(ContainSubstring("func (s *OrderService) DeleteOrder(ctx context.Context, req *DeleteOrderRequest) error"))
	})

	It("generates a real EventPublisher interface and a working no-op default", func() {
		src, err := os.ReadFile(filepath.Join(portsDir, "event_publisher.go"))
		Expect(err).NotTo(HaveOccurred())
		s := string(src)

		Expect(s).To(ContainSubstring("type EventPublisher interface"))
		Expect(s).To(ContainSubstring("Publish(ctx context.Context, events ...event.DomainEvent) error"))
		Expect(s).To(ContainSubstring("type NoopEventPublisher struct{}"))
	})

	It("never overwrites a hand-written service.go, and appends a stub for a newly added RPC without touching existing methods", func() {
		ownedPath := filepath.Join(appDir, "service.go")

		handWritten := `package order

import (
	"context"

	domain "demo/internal/domain/order"
	ports "demo/internal/application/ports"
)

type OrderService struct {
	repo      domain.OrderRepository
	publisher ports.EventPublisher
}

func NewOrderService(repo domain.OrderRepository, publisher ports.EventPublisher) *OrderService {
	return &OrderService{repo: repo, publisher: publisher}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*domain.Order, error) {
	return &domain.Order{Id: "hand-written"}, nil
}

func (s *OrderService) GetOrder(ctx context.Context, req *GetOrderRequest) (*domain.Order, error) {
	panic("sgo: TODO implement GetOrder")
}

func (s *OrderService) ListOrders(ctx context.Context, req *ListOrdersRequest) ([]*domain.Order, error) {
	panic("sgo: TODO implement ListOrders")
}

func (s *OrderService) DeleteOrder(ctx context.Context, req *DeleteOrderRequest) error {
	panic("sgo: TODO implement DeleteOrder")
}
`
		Expect(os.WriteFile(ownedPath, []byte(handWritten), 0644)).To(Succeed())

		// simulate a new RPC (ArchiveOrder) added to the proto
		newSvc := file.Services[0]
		newSvc.Methods = append(newSvc.Methods, sgoproto.Method{Name: "ArchiveOrder", Input: "CreateOrderRequest", Output: "Order"})
		file.Services[0] = newSvc

		fd2, err := sgoproto.Compile(protoDir, "order.proto")
		Expect(err).NotTo(HaveOccurred())

		Expect(core.GenerateApplicationService(file, fd2, p, appDir)).To(Succeed())

		after, err := os.ReadFile(ownedPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(after)).To(ContainSubstring(`return &domain.Order{Id: "hand-written"}, nil`))
		Expect(string(after)).To(ContainSubstring("func (s *OrderService) ArchiveOrder"))
	})

	It("compiles as real Go code, wiring the aggregate repository, event publisher, and application service together end to end", func() {
		modDir := filepath.Join(root, "gobuild")
		Expect(os.MkdirAll(modDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module demo\n\ngo 1.25.0\n"), 0644)).To(Succeed())

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
		copyDir(portsDir, "internal/application/ports")
		copyDir(appDir, "internal/application/order")

		mainSrc := `package main

import (
	"context"
	"fmt"

	domain "demo/internal/domain/order"
	app "demo/internal/application/order"
	"demo/internal/application/ports"
)

func main() {
	svc := app.NewOrderService(nil, ports.NoopEventPublisher{})
	_ = svc
	var _ domain.OrderRepository
	fmt.Println("wired ok")
	_ = context.Background()
}
`
		Expect(os.WriteFile(filepath.Join(modDir, "main.go"), []byte(mainSrc), 0644)).To(Succeed())

		cmd := exec.Command("go", "build", "./...")
		cmd.Dir = modDir
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))
	})
})
