package proto_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

var _ = Describe("GenerateStub", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-proto-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	})

	Context("when the proto file doesn't exist yet", func() {
		It("writes a starter proto with the service and CRUD messages", func() {
			Expect(proto.GenerateStub(dir, "user", "demo")).To(Succeed())

			content, err := os.ReadFile(filepath.Join(dir, "user.proto"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("service UserService"))
			Expect(string(content)).To(ContainSubstring("message User {"))
			Expect(string(content)).To(ContainSubstring(`go_package = "demo/contract/gen/user"`))
		})
	})

	Context("when the proto file already exists", func() {
		It("refuses to overwrite it", func() {
			Expect(proto.GenerateStub(dir, "user", "demo")).To(Succeed())

			err := proto.GenerateStub(dir, "user", "demo")

			Expect(err).To(MatchError(ContainSubstring("already exists")))
		})
	})
})

var _ = Describe("Compile and Build", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-proto-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		Expect(proto.GenerateStub(dir, "user", "demo")).To(Succeed())
	})

	It("compiles the generated stub without error", func() {
		_, err := proto.Compile(dir, "user.proto")
		Expect(err).NotTo(HaveOccurred())
	})

	It("builds an IR whose User message matches the proto fields", func() {
		fd, err := proto.Compile(dir, "user.proto")
		Expect(err).NotTo(HaveOccurred())

		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		Expect(file.Package).To(Equal("user.v1"))

		msg := file.FindMessage("User")
		Expect(msg).NotTo(BeNil())
		Expect(msg.Fields).To(ConsistOf(
			proto.Field{Name: "id", GoName: "Id", Kind: proto.KindString},
			proto.Field{Name: "name", GoName: "Name", Kind: proto.KindString},
			proto.Field{Name: "description", GoName: "Description", Kind: proto.KindString},
		))
	})

	It("builds an IR whose service methods match the proto RPCs", func() {
		fd, err := proto.Compile(dir, "user.proto")
		Expect(err).NotTo(HaveOccurred())

		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		Expect(file.Services).To(HaveLen(1))
		svc := file.Services[0]
		Expect(svc.Name).To(Equal("UserService"))
		Expect(svc.Methods).To(ContainElement(proto.Method{
			Name: "GetUser", Input: "GetUserRequest", Output: "UserResponse",
		}))
		Expect(svc.Methods).To(HaveLen(5))
	})

	It("builds a repeated message field correctly", func() {
		fd, err := proto.Compile(dir, "user.proto")
		Expect(err).NotTo(HaveOccurred())

		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		msg := file.FindMessage("ListUsersResponse")
		Expect(msg).NotTo(BeNil())

		var usersField *proto.Field
		for i := range msg.Fields {
			if msg.Fields[i].Name == "users" {
				usersField = &msg.Fields[i]
			}
		}
		Expect(usersField).NotTo(BeNil())
		Expect(usersField.Kind).To(Equal(proto.KindMessage))
		Expect(usersField.Repeated).To(BeTrue())
		Expect(usersField.MessageType).To(Equal("User"))
		Expect(usersField.GoType()).To(Equal("[]*User"))
	})
})
