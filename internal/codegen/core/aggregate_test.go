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
		Expect(src).To(ContainSubstring("events []event.DomainEvent"))
		Expect(src).To(ContainSubstring("func (a *Order) PullEvents() []event.DomainEvent"))
		Expect(src).To(ContainSubstring(`import event "demo/internal/domain/event"`))
		Expect(src).To(ContainSubstring("type Money struct"))
		Expect(src).To(ContainSubstring("type OrderPlacedEvent struct"))
		Expect(src).To(ContainSubstring(`func (OrderPlacedEvent) EventName() string { return "OrderPlacedEvent" }`))
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

	// buildProject assembles a throwaway module named "demo" (matching
	// p.Module, since the generated files' own import paths are
	// "demo/..." literals) containing the shared event kernel plus
	// whichever domainDir files the caller lists, for a real go
	// build/run to compile against.
	buildProject := func(modName string, files ...string) string {
		GinkgoHelper()
		modDir := filepath.Join(root, modName)
		Expect(os.MkdirAll(modDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module demo\n\ngo 1.25.0\n"), 0644)).To(Succeed())

		eventDir := filepath.Join(modDir, "internal", "domain", "event")
		Expect(os.MkdirAll(eventDir, 0755)).To(Succeed())
		Expect(core.GenerateEventKernel(eventDir)).To(Succeed())

		orderPkgDir := filepath.Join(modDir, "internal", "domain", "order")
		Expect(os.MkdirAll(orderPkgDir, 0755)).To(Succeed())
		for _, name := range files {
			data, err := os.ReadFile(filepath.Join(domainDir, name))
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(filepath.Join(orderPkgDir, name), data, 0644)).To(Succeed())
		}
		return modDir
	}

	It("compiles as real Go code, with a hand-written aggregate method actually using the generated events field", func() {
		ownedPath := filepath.Join(domainDir, "aggregate.go")
		Expect(os.WriteFile(ownedPath, []byte(`package order

func (a *Order) MarkPlaced() {
	a.events = append(a.events, OrderPlacedEvent{OrderId: a.Id})
}
`), 0644)).To(Succeed())

		modDir := buildProject("gobuild", "order_gen.go", "aggregate.go")

		mainSrc := `package main

import "demo/internal/domain/order"

func main() {
	o := &order.Order{Id: "1"}
	o.MarkPlaced()
	events := o.PullEvents()
	if len(events) != 1 {
		panic("expected 1 event")
	}
	if events[0].EventName() != "OrderPlacedEvent" {
		panic("expected OrderPlacedEvent")
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
		Expect(core.GenerateAggregateRepositoryPort(file, fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateDomainErrors(fd, p, domainDir)).To(Succeed())

		repoSrc, err := os.ReadFile(filepath.Join(domainDir, "repository.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(repoSrc)).To(ContainSubstring("type OrderRepository interface"))
		Expect(string(repoSrc)).To(ContainSubstring("Create(ctx context.Context, a *Order) (*Order, error)"))
		Expect(string(repoSrc)).NotTo(ContainSubstring(`"demo/internal/domain/order"`))

		errSrc, err := os.ReadFile(filepath.Join(domainDir, "errors.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(errSrc)).To(ContainSubstring("var ErrNotFound = errors.New"))

		modDir := buildProject("gobuild-repo", "order_gen.go", "aggregate.go", "repository.go", "errors.go")

		cmd := exec.Command("go", "build", "./...")
		cmd.Dir = modDir
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))
	})
})

const sensitiveOrderProto = `syntax = "proto3";

package order.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/order";

message Order {
  option (sgo.aggregate_root) = true;
  string id = 1;
  string customer_id_number = 2 [(sgo.obfuscate_visible) = 3];
  string customer_email = 3 [(sgo.pii) = true];
  Money total = 4;
}

message Money {
  option (sgo.value_object) = true;
  int64 amount = 1;
  string card_number = 2 [(sgo.obfuscate_visible) = 4];
}
`

var _ = Describe("GenerateAggregate, sensitive fields (ARCHITECTURE.md §22)", func() {
	var (
		root, protoDir, domainDir string
		file                      *sgoproto.File
		fd                        protoreflect.FileDescriptor
		p                         core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-core-aggregate-sensitive-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "order.proto"), []byte(sensitiveOrderProto), 0644)).To(Succeed())

		fd, err = sgoproto.Compile(protoDir, "order.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err = sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		p = core.Paths{Module: "demo", Entity: "order"}
		domainDir = filepath.Join(root, "internal", "domain", "order")

		Expect(core.GenerateAggregate(file, fd, p, domainDir)).To(Succeed())
	})

	It("generates MarshalJSON/LogValue only for the obfuscated aggregate and value object, importing encoding/json, log/slog, and mask", func() {
		content, err := os.ReadFile(filepath.Join(domainDir, "order_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		src := string(content)

		Expect(src).To(ContainSubstring(`"encoding/json"`))
		Expect(src).To(ContainSubstring(`"log/slog"`))
		Expect(src).To(ContainSubstring(`mask "demo/internal/domain/mask"`))
		Expect(src).To(ContainSubstring("func (a *Order) MarshalJSON() ([]byte, error)"))
		Expect(src).To(ContainSubstring("func (a *Order) LogValue() slog.Value"))
		Expect(src).To(ContainSubstring("func (a *Money) MarshalJSON() ([]byte, error)"))
		Expect(src).To(ContainSubstring("mask.Obfuscate(a.CustomerIdNumber, 3)"))
		Expect(src).To(ContainSubstring("mask.Obfuscate(a.CardNumber, 4)"))
		// customer_email is PII-only (no obfuscate_visible): logged as-is,
		// no masking call for it specifically.
		Expect(src).To(ContainSubstring(`slog.Any("customer_email", a.CustomerEmail)`))
	})

	// buildSensitiveProject mirrors buildProject above, plus the mask
	// kernel this proto's obfuscated fields need.
	buildSensitiveProject := func(modName string) string {
		GinkgoHelper()
		modDir := filepath.Join(root, modName)
		Expect(os.MkdirAll(modDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module demo\n\ngo 1.25.0\n"), 0644)).To(Succeed())

		eventDir := filepath.Join(modDir, "internal", "domain", "event")
		Expect(os.MkdirAll(eventDir, 0755)).To(Succeed())
		Expect(core.GenerateEventKernel(eventDir)).To(Succeed())

		maskDir := filepath.Join(modDir, "internal", "domain", "mask")
		Expect(os.MkdirAll(maskDir, 0755)).To(Succeed())
		Expect(core.GenerateMaskKernel(maskDir)).To(Succeed())

		orderPkgDir := filepath.Join(modDir, "internal", "domain", "order")
		Expect(os.MkdirAll(orderPkgDir, 0755)).To(Succeed())
		for _, name := range []string{"order_gen.go", "aggregate.go"} {
			data, err := os.ReadFile(filepath.Join(domainDir, name))
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(filepath.Join(orderPkgDir, name), data, 0644)).To(Succeed())
		}
		return modDir
	}

	It("masks in JSON output without mutating the real struct, and redacts obfuscated fields (only) from a real slog line", func() {
		modDir := buildSensitiveProject("gobuild-sensitive")

		mainSrc := `package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"

	"demo/internal/domain/order"
)

func main() {
	o := &order.Order{Id: "1", CustomerIdNumber: "123456789", CustomerEmail: "a@b.com"}

	data, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}
	js := string(data)
	if !strings.Contains(js, "123*****") {
		panic("expected masked value in JSON output, got: " + js)
	}
	if strings.Contains(js, "123456789") {
		panic("real value leaked into JSON output: " + js)
	}

	// The real struct is never mutated by marshaling it — proves the
	// non-destructive shadow-struct approach, not just that masking
	// happened somewhere.
	if o.CustomerIdNumber != "123456789" {
		panic("MarshalJSON mutated the real field: " + o.CustomerIdNumber)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	logger.Info("order", "order", o)
	logLine := buf.String()
	if strings.Contains(logLine, "123456789") {
		panic("real value leaked into log line: " + logLine)
	}
	if !strings.Contains(logLine, "123*****") {
		panic("expected masked value in log line, got: " + logLine)
	}
	if !strings.Contains(logLine, "a@b.com") {
		panic("expected PII-only (unobfuscated) field logged as-is, got: " + logLine)
	}
}
`
		Expect(os.WriteFile(filepath.Join(modDir, "main.go"), []byte(mainSrc), 0644)).To(Succeed())

		cmd := exec.Command("go", "run", ".")
		cmd.Dir = modDir
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))
	})

	It("masks a value object's obfuscated field the same way as the aggregate's", func() {
		modDir := buildSensitiveProject("gobuild-sensitive-vo")

		mainSrc := `package main

import (
	"encoding/json"
	"strings"

	"demo/internal/domain/order"
)

func main() {
	m := &order.Money{Amount: 100, CardNumber: "4111111111111111"}

	data, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	js := string(data)
	if !strings.Contains(js, "4111*****") {
		panic("expected masked card number, got: " + js)
	}
	if strings.Contains(js, "4111111111111111") {
		panic("real card number leaked: " + js)
	}
	if m.CardNumber != "4111111111111111" {
		panic("MarshalJSON mutated the real Money struct")
	}
}
`
		Expect(os.WriteFile(filepath.Join(modDir, "main.go"), []byte(mainSrc), 0644)).To(Succeed())

		cmd := exec.Command("go", "run", ".")
		cmd.Dir = modDir
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))
	})
})
