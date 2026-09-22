package sqlgen_test

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/sqlgen"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

func userFile(protoDir string) (protoreflect.FileDescriptor, *sgoproto.File) {
	GinkgoHelper()

	Expect(sgoproto.GenerateStub(protoDir, "user", "demo")).To(Succeed())

	fd, err := sgoproto.Compile(protoDir, "user.proto")
	Expect(err).NotTo(HaveOccurred())

	file, err := sgoproto.Build(fd)
	Expect(err).NotTo(HaveOccurred())

	return fd, file
}

// requirePostgres skips the calling spec if no Postgres is reachable at
// the codegen's own default connection (127.0.0.1:5432, user/password/db
// "postgres" — see sqlgen's postgres engineDef), so this suite degrades
// gracefully on a machine without a local Postgres rather than failing
// hard. In this development environment, a real `postgresql-16` service
// is running locally and these specs exercise it for real.
func requirePostgres() {
	GinkgoHelper()
	conn, err := net.DialTimeout("tcp", "127.0.0.1:5432", 500*time.Millisecond)
	if err != nil {
		Skip("no local Postgres reachable on 127.0.0.1:5432: " + err.Error())
		return
	}
	conn.Close()
}

var _ = Describe("Generate content", func() {
	var (
		root, protoDir string
		fd             protoreflect.FileDescriptor
		file           *sgoproto.File
		p              core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-sqlgen-content-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		fd, file = userFile(protoDir)
		p = core.Paths{Module: "demo", Entity: "user"}
	})

	It("rejects an unsupported engine", func() {
		destDir := filepath.Join(root, "out")
		err := sqlgen.Generate("oracle", config.PersistenceModeORM, file, fd, p, destDir)
		Expect(err).To(MatchError(ContainSubstring("unsupported SQL engine")))
	})

	It("generates MySQL self-managed code with ? placeholders", func() {
		destDir := filepath.Join(root, "out")
		Expect(sqlgen.Generate(config.PersistenceEngineMySQL, config.PersistenceModeSelfManaged, file, fd, p, destDir)).To(Succeed())

		repoContent, err := os.ReadFile(filepath.Join(destDir, "user_repository_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(repoContent)).To(ContainSubstring("INSERT INTO users (id, name, description) VALUES (?, ?, ?)"))

		connContent, err := os.ReadFile(filepath.Join(destDir, "conn_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(connContent)).To(ContainSubstring(`"github.com/go-sql-driver/mysql"`))
	})

	It("generates MySQL ORM code using gorm.io/driver/mysql", func() {
		destDir := filepath.Join(root, "out")
		Expect(sqlgen.Generate(config.PersistenceEngineMySQL, config.PersistenceModeORM, file, fd, p, destDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(destDir, "conn_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring(`"gorm.io/driver/mysql"`))
		Expect(string(content)).To(ContainSubstring("mysql.Open(dsn)"))
	})

	It("skips message and repeated fields as columns", func() {
		destDir := filepath.Join(root, "out")
		Expect(sqlgen.Generate(config.PersistenceEnginePostgres, config.PersistenceModeORM, file, fd, p, destDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(destDir, "user_repository_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		// ListUsersResponse.Users ([]*User) is a repeated message field on
		// a different message — this file is scoped to the User message's
		// own scalar fields (Name, Description) and must not reference it.
		Expect(string(content)).NotTo(ContainSubstring("Users"))
	})
})

var _ = Describe("Generate with repository_query methods", func() {
	const content = `syntax = "proto3";

package user.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/user";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (UserResponse);

  rpc FindUserByEmail(FindUserByEmailRequest) returns (UserResponse) {
    option (sgo.repository_query) = true;
  }
}

message User {
  string id = 1;
  string name = 2;
  string email = 3;
}

message CreateUserRequest {
  string name = 1;
}

message FindUserByEmailRequest {
  string email = 1;
}

message UserResponse {
  User user = 1;
}
`

	var (
		root, destDir string
		fd            protoreflect.FileDescriptor
		file          *sgoproto.File
		p             core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-sqlgen-query-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir := filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(content), 0644)).To(Succeed())

		fd, err = sgoproto.Compile(protoDir, "user.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err = sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		p = core.Paths{Module: "demo", Entity: "user"}
		destDir = filepath.Join(root, "out")
	})

	It("writes an owned companion file with a panic stub, since SQL query logic can't be auto-generated", func() {
		Expect(sqlgen.Generate(config.PersistenceEnginePostgres, config.PersistenceModeORM, file, fd, p, destDir)).To(Succeed())

		genContent, err := os.ReadFile(filepath.Join(destDir, "user_repository_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(genContent)).NotTo(ContainSubstring("FindByEmail"), "sqlgen never auto-implements — that's memgen-only")

		ownedContent, err := os.ReadFile(filepath.Join(destDir, "user_repository.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(ownedContent)).To(ContainSubstring("func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error)"))
		Expect(string(ownedContent)).To(ContainSubstring(`panic("sgo: TODO implement FindByEmail")`))
	})

	It("survives a hand-written owned-stub implementation across a second generate", func() {
		Expect(sqlgen.Generate(config.PersistenceEnginePostgres, config.PersistenceModeORM, file, fd, p, destDir)).To(Succeed())

		ownedPath := filepath.Join(destDir, "user_repository.go")
		handWritten := `package postgres

import (
	"context"
	"fmt"

	user "demo/internal/domain/user"
)

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return nil, fmt.Errorf("hand-written: %s", email)
}
`
		Expect(os.WriteFile(ownedPath, []byte(handWritten), 0644)).To(Succeed())

		Expect(sqlgen.Generate(config.PersistenceEnginePostgres, config.PersistenceModeORM, file, fd, p, destDir)).To(Succeed())

		after, err := os.ReadFile(ownedPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(after)).To(ContainSubstring("hand-written:"), "the hand-written body must survive regeneration untouched")
	})
})

// These specs build a throwaway module with the generated domain +
// adapter code and actually run it against a real, locally running
// Postgres — not just pattern-matching generated text — the same
// standard Phase 3's httpgen/memgen suites hold generated code to.
var _ = Describe("generated Postgres adapters, running for real", func() {
	BeforeEach(requirePostgres)

	DescribeTable("full CRUD round-trips against a real database",
		func(mode config.PersistenceMode) {
			moduleDir := buildTestModule(config.PersistenceEnginePostgres, mode)
			dropUsersTable()

			out := runInModule(moduleDir, `
				ctx := context.Background()
				if err := postgres.AutoMigrate(`+migrateArgs(mode)+`); err != nil {
					panic(err)
				}

				repo := postgres.NewUserRepository(db)

				created, err := repo.Create(ctx, &user.User{Name: "Ada", Description: "engineer"})
				if err != nil { panic(err) }
				if created.Id == "" { panic("expected a generated id") }

				got, err := repo.Get(ctx, created.Id)
				if err != nil { panic(err) }
				fmt.Printf("got: %s %s\n", got.Name, got.Description)

				grace, err := repo.Create(ctx, &user.User{Name: "Grace", Description: "admiral"})
				if err != nil { panic(err) }

				items, total, err := repo.List(ctx, 0, 0)
				if err != nil { panic(err) }
				fmt.Printf("list: total=%d len=%d\n", total, len(items))

				created.Name = "Ada L"
				updated, err := repo.Update(ctx, created)
				if err != nil { panic(err) }
				fmt.Printf("updated: %s\n", updated.Name)

				if err := repo.Delete(ctx, created.Id); err != nil { panic(err) }
				if err := repo.Delete(ctx, grace.Id); err != nil { panic(err) }

				_, err = repo.Get(ctx, created.Id)
				fmt.Printf("get after delete err is nil: %v\n", err == nil)
			`, connectImport(mode))

			Expect(out).To(ContainSubstring("got: Ada engineer"))
			Expect(out).To(ContainSubstring("list: total=2 len=2"), "regression check: List with page=0,pageSize=0 must return every record")
			Expect(out).To(ContainSubstring("updated: Ada L"))
			Expect(out).To(ContainSubstring("get after delete err is nil: false"))
		},
		Entry("ORM mode", config.PersistenceModeORM),
		Entry("self-managed mode", config.PersistenceModeSelfManaged),
	)
})

func migrateArgs(mode config.PersistenceMode) string {
	if mode == config.PersistenceModeORM {
		return "db"
	}
	return "ctx, db"
}

func connectImport(mode config.PersistenceMode) string {
	if mode == config.PersistenceModeORM {
		return "db, err := postgres.Connect()\n\t\t\t\tif err != nil { panic(err) }"
	}
	return "db, err := postgres.Connect(context.Background())\n\t\t\t\tif err != nil { panic(err) }"
}

func buildTestModule(engine config.PersistenceEngine, mode config.PersistenceMode) string {
	GinkgoHelper()

	dir, err := os.MkdirTemp("", "sgo-sqlgen-module-*")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

	Expect(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\ngo 1.22\n"), 0644)).To(Succeed())

	protoDir := filepath.Join(dir, "contract", "pb")
	fd, file := userFile(protoDir)
	p := core.Paths{Module: "demo", Entity: "user"}

	eventDir := filepath.Join(dir, "internal", "domain", "event")
	Expect(core.GenerateEventKernel(eventDir)).To(Succeed())
	domainDir := filepath.Join(dir, "internal", "domain", "user")
	Expect(core.GenerateAggregate(file, fd, p, domainDir)).To(Succeed())

	adapterDir := filepath.Join(dir, "internal", "infrastructure", "persistence", string(engine))
	Expect(sqlgen.Generate(engine, mode, file, fd, p, adapterDir)).To(Succeed())

	return dir
}

// runInModule writes body into a main() in moduleDir — prefixed with
// connectSnippet, which must declare `db` — and runs it, returning
// combined stdout+stderr.
func runInModule(moduleDir, body, connectSnippet string) string {
	GinkgoHelper()

	main := "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\n\tuser \"demo/internal/domain/user\"\n\tpostgres \"demo/internal/infrastructure/persistence/postgres\"\n)\n\nfunc main() {\n" +
		"\t\t\t\t" + connectSnippet + "\n" + body + "\n}\n"

	mainDir := filepath.Join(moduleDir, "cmd", "harness")
	Expect(os.MkdirAll(mainDir, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(mainDir, "main.go"), []byte(main), 0644)).To(Succeed())

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = moduleDir
	tidyOut, err := tidy.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), string(tidyOut))

	cmd := exec.Command("go", "run", "./cmd/harness")
	cmd.Dir = moduleDir
	out, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), string(out))

	return string(out)
}

func dropUsersTable() {
	GinkgoHelper()
	// Cleanup between spec runs — the "users" table is shared across
	// Entry() runs since both use the same local database, and
	// AutoMigrate only creates a table, it doesn't reset one that's
	// already there from a previous run. This must not fail silently: a
	// swallowed error here (e.g. the local "postgres" role's password
	// doesn't match PGPASSWORD below) leaves stale rows that make later
	// assertions fail with a confusing row-count mismatch instead of a
	// clear connection error.
	cmd := exec.Command("psql", "-h", "127.0.0.1", "-U", "postgres", "-d", "postgres", "-c", "DROP TABLE IF EXISTS users;")
	cmd.Env = append(os.Environ(), "PGPASSWORD=postgres")
	out, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "cleanup of the \"users\" table failed, results would be unreliable: "+string(out))
}
