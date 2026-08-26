package steps

import (
	"context"
	"fmt"
	"os"

	"ilmango/internal/installer"
	"ilmango/internal/run"
)

// deviceGroups are the groups Illogical-mango needs its user in:
//
//	video  — backlight control
//	input  — ydotool's virtual input device
//	i2c    — external monitor brightness over DDC
var deviceGroups = []string{"video", "input", "i2c"}

// userServices are enabled in the user's systemd session. They are user-level
// on purpose: nothing here needs to reach outside the login session.
var userServices = []string{"ydotool"}

type systemStep struct{ base }

func newSystemStep() installer.Step {
	return systemStep{base{
		id:     "system",
		title:  "Configure the system",
		detail: "Group membership for backlight and input, plus user services.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptSystemSetup)
		},
	}}
}

func (s systemStep) Run(ctx context.Context, env *installer.Env) error {
	if err := s.ensureGroups(ctx, env); err != nil {
		return err
	}
	return s.enableServices(ctx, env)
}

// ensureGroups adds the user to the device groups, creating i2c if the kernel
// module package did not.
func (s systemStep) ensureGroups(ctx context.Context, env *installer.Env) error {
	user := os.Getenv("USER")
	if user == "" {
		env.Note("Could not determine the current user, so group membership was left alone. Add yourself to video, input and i2c by hand.")
		return nil
	}

	env.Detail("group membership")

	// groupadd fails harmlessly when the group exists; ask first so the
	// transcript does not show a spurious error.
	if _, err := env.Runner.Output(ctx, run.Command{Name: "getent", Args: []string{"group", "i2c"}}); err != nil {
		if err := env.Runner.Run(ctx, run.Command{Name: "groupadd", Args: []string{"i2c"}, Privileged: true}); err != nil {
			env.Log("could not create the i2c group: " + err.Error())
		}
	}

	for _, group := range deviceGroups {
		cmd := run.Command{Name: "usermod", Args: []string{"-aG", group, user}, Privileged: true}
		if err := env.Runner.Run(ctx, cmd); err != nil {
			return fmt.Errorf("adding %s to the %s group: %w", user, group, err)
		}
	}

	env.NoteApplied("You were added to the video, input and i2c groups. That only takes effect after a full logout.")
	return nil
}

// enableServices starts the user services Illogical-mango relies on.
func (s systemStep) enableServices(ctx context.Context, env *installer.Env) error {
	if !run.Exists("systemctl") {
		env.Note("systemd is not in use here, so no services were enabled. Start ydotoold yourself if you want key remapping.")
		return nil
	}

	env.Detail("user services")
	if err := env.Runner.Run(ctx, run.Command{Name: "systemctl", Args: []string{"--user", "daemon-reload"}}); err != nil {
		env.Log("could not reload the user systemd manager: " + err.Error())
	}

	for _, service := range userServices {
		cmd := run.Command{Name: "systemctl", Args: []string{"--user", "enable", "--now", service}}
		if err := env.Runner.Run(ctx, cmd); err != nil {
			// A missing unit is not fatal: the matching package may have been
			// skipped, and the shell degrades without it.
			env.Log("could not enable " + service + ": " + err.Error())
			env.Note(fmt.Sprintf("The %s service could not be enabled. Enable it later with: systemctl --user enable --now %s", service, service))
		}
	}
	return nil
}
