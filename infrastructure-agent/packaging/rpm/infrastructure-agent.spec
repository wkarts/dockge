Name: infrastructure-agent
Version: %{?agent_version}%{!?agent_version:0.1.0}
Release: 1%{?dist}
Summary: Generic REST infrastructure enrollment and deployment agent
License: MIT
BuildArch: %{_arch}
Requires: ca-certificates
Requires: curl

%description
Generic infrastructure agent for scoped, outbound-only REST Control Plane integration.
Includes an optional interactive host bootstrap that can preserve existing
Docker/Dockge installations or prepare a new API-first Dockge in coexistence.

%install
mkdir -p \
  %{buildroot}/usr/bin \
  %{buildroot}/usr/lib/infrastructure-agent \
  %{buildroot}/etc/infrastructure-agent \
  %{buildroot}/usr/lib/systemd/system \
  %{buildroot}/var/lib/infrastructure-agent
install -m 0755 %{_sourcedir}/infra-agent %{buildroot}/usr/bin/infra-agent
install -m 0755 %{_sourcedir}/infra-agent-installer %{buildroot}/usr/bin/infra-agent-installer
install -m 0644 %{_sourcedir}/bootstrap-host.sh %{buildroot}/usr/lib/infrastructure-agent/bootstrap-host.sh
install -m 0644 %{_sourcedir}/agent.json.example %{buildroot}/etc/infrastructure-agent/agent.json.example
install -m 0644 %{_sourcedir}/infrastructure-agent.service %{buildroot}/usr/lib/systemd/system/infrastructure-agent.service

%files
/usr/bin/infra-agent
/usr/bin/infra-agent-installer
/usr/lib/infrastructure-agent/bootstrap-host.sh
/etc/infrastructure-agent/agent.json.example
/usr/lib/systemd/system/infrastructure-agent.service
%dir /var/lib/infrastructure-agent

%post
mkdir -p /etc/infrastructure-agent/secrets /var/lib/infrastructure-agent
chmod 0750 /etc/infrastructure-agent >/dev/null 2>&1 || true
chmod 0700 /etc/infrastructure-agent/secrets /var/lib/infrastructure-agent >/dev/null 2>&1 || true
systemctl daemon-reload >/dev/null 2>&1 || true
systemctl enable infrastructure-agent.service >/dev/null 2>&1 || true
printf '\n╔══════════════════════════════════════════════════════════════╗\n'
printf '║ Generic Infrastructure Agent instalado                     ║\n'
printf '╚══════════════════════════════════════════════════════════════╝\n'
printf 'Próximo passo: sudo infra-agent-installer\n'
printf 'O assistente pode preservar runtime atual ou preparar Docker/Dockge.\n\n'

%preun
if [ "$1" = "0" ]; then systemctl disable --now infrastructure-agent.service >/dev/null 2>&1 || true; fi
