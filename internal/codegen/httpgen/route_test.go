package httpgen_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/httpgen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

func userFile(protoDir string) *sgoproto.File {
	GinkgoHelper()

	Expect(sgoproto.GenerateStub(protoDir, "user", "demo")).To(Succeed())

	fd, err := sgoproto.Compile(protoDir, "user.proto")
	Expect(err).NotTo(HaveOccurred())

	file, err := sgoproto.Build(fd)
	Expect(err).NotTo(HaveOccurred())

	return file
}

func routeFor(routes []httpgen.Route, rpc string) httpgen.Route {
	GinkgoHelper()

	for _, r := range routes {
		if r.Method.Name == rpc {
			return r
		}
	}
	Fail("no route derived for RPC " + rpc)
	return httpgen.Route{}
}

var _ = Describe("BuildRoutes", func() {
	var (
		dir    string
		file   *sgoproto.File
		routes []httpgen.Route
	)

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-httpgen-route-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		file = userFile(filepath.Join(dir, "contract", "pb"))
		routes = httpgen.BuildRoutes(file, "user")
	})

	It("derives one route per RPC", func() {
		Expect(routes).To(HaveLen(5))
	})

	It("maps CreateUser to POST /api/v1/users with a body and no id", func() {
		r := routeFor(routes, "CreateUser")
		Expect(r.Verb).To(Equal("POST"))
		Expect(r.BasePath).To(Equal("/api/v1/users"))
		Expect(r.HasBody).To(BeTrue())
		Expect(r.HasID).To(BeFalse())
		Expect(r.FullPath(":id")).To(Equal("/api/v1/users"))
	})

	It("maps GetUser to GET /api/v1/users/:id with an id and no body", func() {
		r := routeFor(routes, "GetUser")
		Expect(r.Verb).To(Equal("GET"))
		Expect(r.HasID).To(BeTrue())
		Expect(r.HasBody).To(BeFalse())
		Expect(r.FullPath(":id")).To(Equal("/api/v1/users/:id"))
		Expect(r.FullPath("{id}")).To(Equal("/api/v1/users/{id}"))
	})

	It("maps ListUsers to GET /api/v1/users with no id and no body", func() {
		r := routeFor(routes, "ListUsers")
		Expect(r.Verb).To(Equal("GET"))
		Expect(r.HasID).To(BeFalse())
		Expect(r.HasBody).To(BeFalse())
	})

	It("maps UpdateUser to PUT /api/v1/users/:id with an id and a body", func() {
		r := routeFor(routes, "UpdateUser")
		Expect(r.Verb).To(Equal("PUT"))
		Expect(r.HasID).To(BeTrue())
		Expect(r.HasBody).To(BeTrue())
	})

	It("maps DeleteUser to DELETE /api/v1/users/:id with an id and no body", func() {
		r := routeFor(routes, "DeleteUser")
		Expect(r.Verb).To(Equal("DELETE"))
		Expect(r.HasID).To(BeTrue())
		Expect(r.HasBody).To(BeFalse())
	})

	Context("when an RPC doesn't match the CRUD naming convention", func() {
		It("falls back to POST, keyed by id if the input has one", func() {
			custom := &sgoproto.File{
				Messages: file.Messages,
				Services: []sgoproto.Service{{
					Name: "UserService",
					Methods: []sgoproto.Method{
						{Name: "ArchiveUser", Input: "DeleteUserRequest", Output: "DeleteUserResponse"},
					},
				}},
			}

			routes := httpgen.BuildRoutes(custom, "user")
			Expect(routes).To(HaveLen(1))
			Expect(routes[0].Verb).To(Equal("POST"))
			Expect(routes[0].HasID).To(BeTrue(), "DeleteUserRequest has an id field")
			Expect(routes[0].HasBody).To(BeTrue())
		})
	})
})
