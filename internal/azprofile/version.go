package azprofile

// Version is overridden at build time via:
//   go build -ldflags "-X 'github.com/neverprepared/azprofile/internal/azprofile.Version=v1.3.2'"
// The release workflow injects the git tag; local `make build` leaves it "dev".
var Version = "dev"
