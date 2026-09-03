Name: infrastructure-agent
Version: %{?agent_version}%{!?agent_version:0.1.0}
Release: 1%{?dist}
Summary: Generic REST infrastructure enrollment and deployment agent
License: MIT
BuildArch: %{_arch}
Requires: ca-certificates

%description
Generic infrastructure agent for scoped, outbound-only REST Control Plane integration.

%install
mkdir -p %{buildroot}/usr/bin %{buildroot}/etc/infrastructure-agent %{buildroot}/usr/lib/systemd/system %{buildroot}/var/lib/infrastructure-agent
install -m 0755 %{_sourcedir}/infra-agent %{buildroot}/usr/bin/infra-agent
install -m 0755 %{_sourcedir}/infra-agent-installer %{buildroot}/usr/bin/infra-agent-installer
install -m 0644 %{_sourcedir}/agent.json.example %{buildroot}/etc/infrastructure-agent/agent.json.example
install -m 0644 %{_sourcedir}/infrastructure-agent.service %{buildroot}/usr/lib/systemd/system/infrastructure-agent.service

%files
/usr/bin/infra-agent
/usr/bin/infra-agent-installer
/etc/infrastructure-agent/agent.json.example
/usr/lib/systemd/system/infrastructure-agent.service
%dir /var/lib/infrastructure-agent

%post
mkdir -p /etc/infrastructure-agent/secrets /var/lib/infrastructure-agent
chmod 0750 /etc/infrastructure-agent >/dev/null 2>&1 || true
chmod 0700 /etc/infrastructure-agent/secrets /var/lib/infrastructure-agent >/dev/null 2>&1 || true
systemctl daemon-reload >/dev/null 2>&1 || true
systemctl enable infrastructure-agent.service >/dev/null 2>&1 || true
printf '\nGeneric Infrastructure Agent instalado. Execute: sudo infra-agent-installer\n\n'

%preun
if [ "$1" = "0" ]; then systemctl disable --now infrastructure-agent.service >/dev/null 2>&1 || true; fi
