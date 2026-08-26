# Paths are anchored to the checkout rather than the working directory, so this
# Makefile works whether it is invoked as `make -f packaging/Makefile` from the
# root or as `make -C packaging`.
TOP := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))/..
SRC := $(TOP)/src
GOSRC := $(TOP)/installer

PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin
SHAREDIR = $(PREFIX)/share
APPLICATIONS_DIR = $(SHAREDIR)/applications
ICON_DIR = $(SHAREDIR)/icons/hicolor/scalable/apps
SHELL_INSTALL_DIR = $(SHAREDIR)/quickshell/ilmango
DOC_DIR = $(SHAREDIR)/doc/ilmango-shell
SYSTEMD_USER_DIR ?= $(PREFIX)/lib/systemd/user

GO ?= go
INSTALLER_BIN = $(GOSRC)/ilmango-installer

.PHONY: all build installer installer-test test-local install install-installer install-bin install-shell install-systemd install-icon install-desktop install-docs uninstall uninstall-bin uninstall-shell uninstall-systemd uninstall-icon uninstall-desktop uninstall-docs

all: build

# The Go installer. Built on demand rather than as part of `build`, so the
# shell-only workflow keeps working on machines without a Go toolchain.
installer:
	@cd $(GOSRC) && $(GO) build -o $(INSTALLER_BIN) ./cmd/ilmango-installer

installer-test:
	@cd $(GOSRC) && $(GO) vet ./cmd/... ./internal/...
	@cd $(GOSRC) && $(GO) test ./cmd/... ./internal/...

build:
	@chmod +x $(SRC)/scripts/ilmango
	@chmod +x $(SRC)/scripts/test-local-distribution.sh
	@chmod +x $(TOP)/setup
	@find $(SRC)/scripts -type f \( -name "*.sh" -o -name "*.fish" -o -name "*.py" \) -exec chmod +x {} +

test-local: build
	@bash $(SRC)/scripts/test-local-distribution.sh

install-bin:
	@install -Dm755 $(SRC)/scripts/ilmango $(BINDIR)/ilmango

install-shell:
	@mkdir -p $(SHELL_INSTALL_DIR)
	@for file in $(SRC)/*.qml; do install -Dm644 "$$file" "$(SHELL_INSTALL_DIR)/$$(basename $$file)"; done
	@install -Dm644 $(TOP)/docs/CHANGELOG.md $(SHELL_INSTALL_DIR)/CHANGELOG.md
	@while IFS= read -r file; do \
		[ -n "$$file" ] || continue; \
		if [ "$$file" = "setup" ]; then \
			install -Dm755 "$(SRC)/$$file" "$(SHELL_INSTALL_DIR)/$$file"; \
		else \
			install -Dm644 "$(SRC)/$$file" "$(SHELL_INSTALL_DIR)/$$file"; \
		fi; \
	done < $(SRC)/sdata/runtime-root-files.txt
	@while IFS= read -r dir; do \
		[ -n "$$dir" ] || continue; \
		mkdir -p "$(SHELL_INSTALL_DIR)/$$dir"; \
		cp -a "$(SRC)/$$dir"/. "$(SHELL_INSTALL_DIR)/$$dir/"; \
	done < $(SRC)/sdata/runtime-payload-dirs.txt
	@# Strip local AI/agent directives — never distributed (cp -a above copies them in)
	@find $(SHELL_INSTALL_DIR) \( -name AGENTS.md -o -name CLAUDE.md -o -name CODEX.md -o -name PI.md -o -name codemap.md -o -name .mcp.json -o -name opencode.json -o -name skills-lock.json \) -delete
	@find $(SHELL_INSTALL_DIR) \( -name .agents -o -name .claude -o -name .codex -o -name .factory -o -name .opencode -o -name .codebase-memory -o -name .impeccable -o -name .pi-subagents \) -type d -prune -exec rm -rf {} +
	@rm -rf $(SHELL_INSTALL_DIR)/assets/graphify-out
	@find $(SHELL_INSTALL_DIR)/assets/images/mascot -maxdepth 1 -type f \( -name 'inir-mascot-*.png' -o -name 'inir-mascot-*.gif' -o -name PROMPTS.md \) -delete
	@rm -rf $(SHELL_INSTALL_DIR)/assets/images/mascot/frames
	@# Strip maintainer and development tooling — useless on a user's machine
	@rm -rf $(SHELL_INSTALL_DIR)/scripts/agents $(SHELL_INSTALL_DIR)/translations/tools $(SHELL_INSTALL_DIR)/translations/l10n
	@rm -f $(SHELL_INSTALL_DIR)/scripts/release.sh $(SHELL_INSTALL_DIR)/scripts/wiki-sync.sh $(SHELL_INSTALL_DIR)/scripts/verify-docs.sh $(SHELL_INSTALL_DIR)/scripts/qml-check.fish $(SHELL_INSTALL_DIR)/scripts/test-local-distribution.sh $(SHELL_INSTALL_DIR)/scripts/test-mascot-pack-flow.sh
	@find $(SHELL_INSTALL_DIR)/scripts -type f \( -name "*.sh" -o -name "*.fish" -o -name "*.py" \) -exec chmod +x {} +
	@printf '{\n  "version": "%s",\n  "commit": "manual",\n  "installed_at": "%s",\n  "installedAt": "%s",\n  "source": "make-install",\n  "repo_path": "",\n  "repoPath": "",\n  "install_mode": "package-managed",\n  "installMode": "package-managed",\n  "update_strategy": "package-manager",\n  "updateStrategy": "package-manager",\n  "package_manager": "manual",\n  "packageManager": "manual",\n  "package_name": "source-install",\n  "packageName": "source-install",\n  "package_update_hint": "sudo make install",\n  "packageUpdateHint": "sudo make install"\n}\n' "$$(cat $(SRC)/VERSION)" "$$(date -Iseconds)" "$$(date -Iseconds)" > $(SHELL_INSTALL_DIR)/version.json

install-systemd:
	@mkdir -p $(SYSTEMD_USER_DIR)
	@sed -e 's|^ExecStart=.*|ExecStart=$(BINDIR)/ilmango run --session|' \
		-e 's|^ExecStopPost=-.*|ExecStopPost=-$(BINDIR)/ilmango cleanup-orphans|' \
		$(SRC)/assets/systemd/ilmango.service > $(SYSTEMD_USER_DIR)/ilmango.service
	@chmod 644 $(SYSTEMD_USER_DIR)/ilmango.service

install-icon:
	@install -Dm644 $(SRC)/assets/icons/desktop-symbolic.svg $(ICON_DIR)/ilmango.svg
	@gtk-update-icon-cache -q $(SHAREDIR)/icons/hicolor 2>/dev/null || true

install-desktop:
	@install -Dm644 $(SRC)/assets/applications/ilmango.desktop $(APPLICATIONS_DIR)/ilmango.desktop
	@install -Dm644 $(SRC)/assets/applications/ilmango-settings.desktop $(APPLICATIONS_DIR)/ilmango-settings.desktop
	@update-desktop-database -q $(APPLICATIONS_DIR) 2>/dev/null || true

install-docs:
	@install -Dm644 README.md $(DOC_DIR)/README.md
	@install -Dm644 $(TOP)/docs/SETUP.md $(DOC_DIR)/SETUP.md
	@install -Dm644 $(TOP)/docs/IPC.md $(DOC_DIR)/IPC.md

install-installer: installer
	@install -Dm755 $(INSTALLER_BIN) $(BINDIR)/ilmango-installer

install: build install-bin install-shell install-systemd install-icon install-desktop install-docs

uninstall-bin:
	@rm -f $(BINDIR)/ilmango $(BINDIR)/ilmango-installer

uninstall-shell:
	@rm -rf $(SHELL_INSTALL_DIR)

uninstall-systemd:
	@rm -f $(SYSTEMD_USER_DIR)/ilmango.service

uninstall-icon:
	@rm -f $(ICON_DIR)/ilmango.svg
	@gtk-update-icon-cache -q $(SHAREDIR)/icons/hicolor 2>/dev/null || true

uninstall-desktop:
	@rm -f $(APPLICATIONS_DIR)/ilmango.desktop $(APPLICATIONS_DIR)/ilmango-settings.desktop
	@update-desktop-database -q $(APPLICATIONS_DIR) 2>/dev/null || true

uninstall-docs:
	@rm -rf $(DOC_DIR)

uninstall: uninstall-systemd uninstall-desktop uninstall-icon uninstall-docs uninstall-shell uninstall-bin
