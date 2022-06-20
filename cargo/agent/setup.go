package agent

func GetServiceInstallationTemplate() string {
	return `
#/etc/systemd/system/svcname.service
[Unit]
Description=cargo service
Requires=snap.docker.dockerd.service
After=snap.docker.dockerd.service

[Service]
Type=oneshot
RemainAfterExit=true
WorkingDirectory=%s
ExecStart=%s
ExecStop=%s stop

[Install]
WantedBy=multi-user.target
	`
}
