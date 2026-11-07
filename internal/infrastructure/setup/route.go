package setup

import "fmt"

// Route is the conversation one run of setup is having. It is read once from the machine and
// decides the screen, its heading, its options and its go-ahead together, so they cannot drift
// apart. Removal is a screen reachable from every route, never a route of its own.
type Route string

const (
	RouteInstall   Route = "install"   // nothing recorded
	RouteUpdate    Route = "update"    // the recorded version is older than this one
	RouteDowngrade Route = "downgrade" // the recorded version is newer than this one
	RouteManage    Route = "manage"    // the versions match: nothing to install
)

// DecideRoute chooses the route from what is recorded and what this setup carries.
func DecideRoute(installed bool, installedVersion, thisVersion string) Route {
	if !installed {
		return RouteInstall
	}
	switch CompareVersions(installedVersion, thisVersion) {
	case -1:
		return RouteUpdate
	case 1:
		return RouteDowngrade
	}
	return RouteManage
}

// Wording is everything a route screen says.
type Wording struct {
	Heading string `json:"heading"`
	Lead    string `json:"lead"`
	Flow    bool   `json:"flow"` // show the "installed to this" version line
	Action  string `json:"action"`
}

// Words returns the wording for a route. One version belongs in a sentence; an update or a
// downgrade is about two, so neither is named in the heading and the flow line carries both.
func Words(r Route, installedVersion, thisVersion string) Wording {
	switch r {
	case RouteUpdate:
		return Wording{Heading: "Update WebVigil", Flow: true, Action: "Update",
			Lead: "A newer WebVigil is ready. Your websites, history and settings are kept."}
	case RouteDowngrade:
		return Wording{Heading: "Go back to an earlier WebVigil", Flow: true, Action: "Go back",
			Lead: "This setup carries an older version than the one installed. Your data is kept; an older " +
				"WebVigil may not open data written by a newer one."}
	case RouteManage:
		return Wording{Heading: "WebVigil is installed", Action: "Repair",
			Lead: fmt.Sprintf("Version %s is installed, the version this setup carries. Repair puts the program "+
				"files back and leaves your websites, history and settings alone.", installedVersion)}
	}
	return Wording{Heading: "Install WebVigil", Action: "Install",
		Lead: fmt.Sprintf("This installs WebVigil %s for your user account only; no administrator rights are needed.", thisVersion)}
}

// Step is one step of an operation with its share of the progress bar.
type Step struct {
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

// The bar is weighted by time, not by step count. The install weights are each step's median
// time in milliseconds over twelve installs recorded in setup.log: the shortcut calls (one
// PowerShell process each) take nearly all of it. No removal has been timed yet, so its weights
// are still estimates; replace them from the step log of a real uninstall.
var (
	InstallSteps = []Step{{"Unpacking WebVigil", 76}, {"Registering with Windows", 11}, {"Creating shortcuts", 1680}, {"Applying settings", 0}}
	RemoveSteps  = []Step{{"Removing shortcuts", 40}, {"Removing registry entries", 10}, {"Removing your data", 20}, {"Removing program files", 30}}
)

// Progress returns the bar position, in percent, once steps before done have finished.
func Progress(steps []Step, done int) int {
	total, sum := 0, 0
	for i, s := range steps {
		total += s.Weight
		if i < done {
			sum += s.Weight
		}
	}
	if total == 0 {
		return 0
	}
	return sum * 100 / total
}
