package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker-language-server/internal/hub"
	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/uri"
)

var topLevelNodes = []protocol.CompletionItem{
	{
		Label:         "configs",
		Documentation: "Configurations that are shared among multiple services.",
	},
	{
		Label:         "include",
		Documentation: "compose sub-projects to be included.",
	},
	{
		Label:         "models",
		Documentation: "Language models that will be used by your application.",
	},
	{
		Label:         "name",
		Documentation: "define the Compose project name, until user defines one explicitly.",
	},
	{
		Label:         "networks",
		Documentation: "Networks that are shared among multiple services.",
	},
	{
		Label:         "secrets",
		Documentation: "Secrets that are shared among multiple services.",
	},
	{
		Label:         "services",
		Documentation: "The services that will be used by your application.",
	},
	{
		Label:         "version",
		Documentation: "declared for backward compatibility, ignored. Please remove it.",
	},
	{
		Label:         "volumes",
		Documentation: "Named volumes that are shared among multiple services.",
	},
}

func serviceProperties(line, character, prefixLength protocol.UInteger, spacing string) []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{
			Label:            "annotations",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
			TextEdit:         textEdit(fmt.Sprintf("annotations:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "attach",
			Detail:           types.CreateStringPointer("boolean or string"),
			TextEdit:         textEdit("attach: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "blkio_config",
			Detail:           types.CreateStringPointer("object"),
			Documentation:    "Block IO configuration for the service.",
			TextEdit:         textEdit(fmt.Sprintf("blkio_config:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "build",
			Detail:           types.CreateStringPointer("object or string"),
			Documentation:    "Configuration options for building the service's image.",
			TextEdit:         textEdit("build:", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cap_add",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Add Linux capabilities. For example, 'CAP_SYS_ADMIN', 'SYS_ADMIN', or 'NET_ADMIN'.",
			TextEdit:         textEdit(fmt.Sprintf("cap_add:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cap_drop",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Drop Linux capabilities. For example, 'CAP_SYS_ADMIN', 'SYS_ADMIN', or 'NET_ADMIN'.",
			TextEdit:         textEdit(fmt.Sprintf("cap_drop:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cgroup",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Specify the cgroup namespace to join. Use 'host' to use the host's cgroup namespace, or 'private' to use a private cgroup namespace.",
			TextEdit:         textEdit("cgroup: ${1|host,private|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cgroup_parent",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Specify an optional parent cgroup for the container.",
			TextEdit:         textEdit("cgroup_parent: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "command",
			Detail:           types.CreateStringPointer("array or null or string"),
			Documentation:    "Command to run in the container, which can be specified as a string (shell form) or array (exec form).",
			TextEdit:         textEdit("command:", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "configs",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Configuration for service configs or secrets, defining how they are mounted in the container.",
			TextEdit:         textEdit(fmt.Sprintf("configs:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "container_name",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Specify a custom container name, rather than a generated default name.",
			TextEdit:         textEdit("container_name: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cpu_count",
			Detail:           types.CreateStringPointer("integer or string"),
			Documentation:    "Number of usable CPUs.",
			TextEdit:         textEdit("cpu_count: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cpu_percent",
			Detail:           types.CreateStringPointer("integer or string"),
			Documentation:    "Percentage of CPU resources to use.",
			TextEdit:         textEdit("cpu_percent: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cpu_period",
			Detail:           types.CreateStringPointer("number or string"),
			Documentation:    "Limit the CPU CFS (Completely Fair Scheduler) period.",
			TextEdit:         textEdit("cpu_period: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cpu_quota",
			Detail:           types.CreateStringPointer("number or string"),
			Documentation:    "Limit the CPU CFS (Completely Fair Scheduler) quota.",
			TextEdit:         textEdit("cpu_quota: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cpu_rt_period",
			Detail:           types.CreateStringPointer("number or string"),
			Documentation:    "Limit the CPU real-time period in microseconds or a duration.",
			TextEdit:         textEdit("cpu_rt_period: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cpu_rt_runtime",
			Detail:           types.CreateStringPointer("number or string"),
			Documentation:    "Limit the CPU real-time runtime in microseconds or a duration.",
			TextEdit:         textEdit("cpu_rt_runtime: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cpu_shares",
			Detail:           types.CreateStringPointer("number or string"),
			Documentation:    "CPU shares (relative weight) for the container.",
			TextEdit:         textEdit("cpu_shares: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cpus",
			Detail:           types.CreateStringPointer("number or string"),
			Documentation:    "Number of CPUs to use. A floating-point value is supported to request partial CPUs.",
			TextEdit:         textEdit("cpus: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cpuset",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "CPUs in which to allow execution (0-3, 0,1).",
			TextEdit:         textEdit("cpuset: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "credential_spec",
			Detail:           types.CreateStringPointer("object"),
			Documentation:    "Configure the credential spec for managed service account.",
			TextEdit:         textEdit(fmt.Sprintf("credential_spec:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "depends_on",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "Express dependency between services. Service dependencies cause services to be started in dependency order. The dependent service will wait for the dependency to be ready before starting.",
			TextEdit:         textEdit(fmt.Sprintf("depends_on:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "deploy",
			Detail:           types.CreateStringPointer("null or object"),
			Documentation:    "Deployment configuration for the service.",
			TextEdit:         textEdit("deploy:", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "develop",
			Detail:           types.CreateStringPointer("null or object"),
			Documentation:    "Development configuration for the service, used for development workflows.",
			TextEdit:         textEdit("develop:", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "device_cgroup_rules",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "A list of unique string values.",
			TextEdit:         textEdit(fmt.Sprintf("device_cgroup_rules:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "devices",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "List of device mappings for the container.",
			TextEdit:         textEdit(fmt.Sprintf("devices:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "dns",
			Detail:           types.CreateStringPointer("array or string"),
			Documentation:    "Either a single string or a list of strings.",
			TextEdit:         textEdit("dns:", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "dns_opt",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Custom DNS options to be passed to the container's DNS resolver.",
			TextEdit:         textEdit(fmt.Sprintf("dns_opt:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "dns_search",
			Detail:           types.CreateStringPointer("array or string"),
			Documentation:    "Either a single string or a list of strings.",
			TextEdit:         textEdit("dns_search:", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "domainname",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Custom domain name to use for the service container.",
			TextEdit:         textEdit("domainname: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "entrypoint",
			Detail:           types.CreateStringPointer("array or null or string"),
			Documentation:    "Command to run in the container, which can be specified as a string (shell form) or array (exec form).",
			TextEdit:         textEdit("entrypoint:", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "env_file",
			Detail:           types.CreateStringPointer("array or string"),
			TextEdit:         textEdit("env_file:", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "environment",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
			TextEdit:         textEdit(fmt.Sprintf("environment:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "expose",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Expose ports without publishing them to the host machine - they'll only be accessible to linked services.",
			TextEdit:         textEdit(fmt.Sprintf("expose:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "extends",
			Detail:           types.CreateStringPointer("object or string"),
			Documentation:    "Extend another service, in the current file or another file.",
			TextEdit:         textEdit("extends:", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "external_links",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Link to services started outside this Compose application. Specify services as <service_name>:<alias>.",
			TextEdit:         textEdit(fmt.Sprintf("external_links:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "extra_hosts",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "Additional hostnames to be defined in the container's /etc/hosts file.",
			TextEdit:         textEdit(fmt.Sprintf("extra_hosts:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "gpus",
			Detail:           types.CreateStringPointer("array or string"),
			TextEdit:         textEdit("gpus:", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "group_add",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Add additional groups which user inside the container should be member of.",
			TextEdit:         textEdit(fmt.Sprintf("group_add:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "healthcheck",
			Detail:           types.CreateStringPointer("object"),
			Documentation:    "Configuration options to determine whether the container is healthy.",
			TextEdit:         textEdit(fmt.Sprintf("healthcheck:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "hostname",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Define a custom hostname for the service container.",
			TextEdit:         textEdit("hostname: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "image",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Specify the image to start the container from. Can be a repository/tag, a digest, or a local image ID.",
			TextEdit:         textEdit("image: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "init",
			Detail:           types.CreateStringPointer("boolean or string"),
			Documentation:    "Run as an init process inside the container that forwards signals and reaps processes.",
			TextEdit:         textEdit("init: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "ipc",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "IPC sharing mode for the service container. Use 'host' to share the host's IPC namespace, 'service:[service_name]' to share with another service, or 'shareable' to allow other services to share this service's IPC namespace.",
			TextEdit:         textEdit("ipc: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "isolation",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Container isolation technology to use. Supported values are platform-specific.",
			TextEdit:         textEdit("isolation: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "label_file",
			Detail:           types.CreateStringPointer("array or string"),
			TextEdit:         textEdit("label_file:", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "labels",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
			TextEdit:         textEdit(fmt.Sprintf("labels:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "links",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Link to containers in another service. Either specify both the service name and a link alias (SERVICE:ALIAS), or just the service name.",
			TextEdit:         textEdit(fmt.Sprintf("links:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "logging",
			Detail:           types.CreateStringPointer("object"),
			Documentation:    "Logging configuration for the service.",
			TextEdit:         textEdit(fmt.Sprintf("logging:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "mac_address",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Container MAC address to set.",
			TextEdit:         textEdit("mac_address: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "mem_limit",
			Detail:           types.CreateStringPointer("number or string"),
			Documentation:    "Memory limit for the container. A string value can use suffix like '2g' for 2 gigabytes.",
			TextEdit:         textEdit("mem_limit: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "mem_reservation",
			Detail:           types.CreateStringPointer("integer or string"),
			Documentation:    "Memory reservation for the container.",
			TextEdit:         textEdit("mem_reservation: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "mem_swappiness",
			Detail:           types.CreateStringPointer("integer or string"),
			Documentation:    "Container memory swappiness as percentage (0 to 100).",
			TextEdit:         textEdit("mem_swappiness: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "memswap_limit",
			Detail:           types.CreateStringPointer("number or string"),
			Documentation:    "Amount of memory the container is allowed to swap to disk. Set to -1 to enable unlimited swap.",
			TextEdit:         textEdit("memswap_limit: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "models",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "AI Models to use, referencing entries under the top-level models key.",
			TextEdit:         textEdit(fmt.Sprintf("models:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "network_mode",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Network mode. Values can be 'bridge', 'host', 'none', 'service:[service name]', or 'container:[container name]'.",
			TextEdit:         textEdit("network_mode: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "networks",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "Networks to join, referencing entries under the top-level networks key. Can be a list of network names or a mapping of network name to network configuration.",
			TextEdit:         textEdit(fmt.Sprintf("networks:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "oom_kill_disable",
			Detail:           types.CreateStringPointer("boolean or string"),
			Documentation:    "Disable OOM Killer for the container.",
			TextEdit:         textEdit("oom_kill_disable: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "oom_score_adj",
			Detail:           types.CreateStringPointer("integer or string"),
			Documentation:    "Tune host's OOM preferences for the container (accepts -1000 to 1000).",
			TextEdit:         textEdit("oom_score_adj: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "pid",
			Detail:           types.CreateStringPointer("null or string"),
			Documentation:    "PID mode for container.",
			TextEdit:         textEdit("pid: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "pids_limit",
			Detail:           types.CreateStringPointer("number or string"),
			Documentation:    "Tune a container's PIDs limit. Set to -1 for unlimited PIDs.",
			TextEdit:         textEdit("pids_limit: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "platform",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Target platform to run on, e.g., 'linux/amd64', 'linux/arm64', or 'windows/amd64'.",
			TextEdit:         textEdit("platform: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "ports",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Expose container ports. Short format ([HOST:]CONTAINER[/PROTOCOL]).",
			TextEdit:         textEdit(fmt.Sprintf("ports:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "post_start",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Commands to run after the container starts. If any command fails, the container stops.",
			TextEdit:         textEdit(fmt.Sprintf("post_start:\n%v      - command:", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "pre_stop",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Commands to run before the container stops. If any command fails, the container stop is aborted.",
			TextEdit:         textEdit(fmt.Sprintf("pre_stop:\n%v      - command:", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "privileged",
			Detail:           types.CreateStringPointer("boolean or string"),
			Documentation:    "Give extended privileges to the service container.",
			TextEdit:         textEdit("privileged: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "profiles",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "A list of unique string values.",
			TextEdit:         textEdit(fmt.Sprintf("profiles:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "provider",
			Detail:           types.CreateStringPointer("object"),
			Documentation:    "Specify a service which will not be manage by Compose directly, and delegate its management to an external provider.",
			TextEdit:         textEdit(fmt.Sprintf("provider:\n%v      type: ${1:model}\n%v      options:\n%v        ${2:model}: ${3:ai/example-model}", spacing, spacing, spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "pull_policy",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Policy for pulling images. Options include: 'always', 'never', 'if_not_present', 'missing', 'build', or time-based refresh policies.",
			TextEdit:         textEdit("pull_policy: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "pull_refresh_after",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Time after which to refresh the image. Used with pull_policy=refresh.",
			TextEdit:         textEdit("pull_refresh_after: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "read_only",
			Detail:           types.CreateStringPointer("boolean or string"),
			Documentation:    "Mount the container's filesystem as read only.",
			TextEdit:         textEdit("read_only: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "restart",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Restart policy for the service container. Options include: 'no', 'always', 'on-failure', and 'unless-stopped'.",
			TextEdit:         textEdit("restart: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "runtime",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Runtime to use for this container, e.g., 'runc'.",
			TextEdit:         textEdit("runtime: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "scale",
			Detail:           types.CreateStringPointer("integer or string"),
			Documentation:    "Number of containers to deploy for this service.",
			TextEdit:         textEdit("scale: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "secrets",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Configuration for service configs or secrets, defining how they are mounted in the container.",
			TextEdit:         textEdit(fmt.Sprintf("secrets:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "security_opt",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Override the default labeling scheme for each container.",
			TextEdit:         textEdit(fmt.Sprintf("security_opt:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "shm_size",
			Detail:           types.CreateStringPointer("number or string"),
			Documentation:    "Size of /dev/shm. A string value can use suffix like '2g' for 2 gigabytes.",
			TextEdit:         textEdit("shm_size: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "stdin_open",
			Detail:           types.CreateStringPointer("boolean or string"),
			Documentation:    "Keep STDIN open even if not attached.",
			TextEdit:         textEdit("stdin_open: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "stop_grace_period",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Time to wait for the container to stop gracefully before sending SIGKILL (e.g., '1s', '1m30s').",
			TextEdit:         textEdit("stop_grace_period: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "stop_signal",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Signal to stop the container (e.g., 'SIGTERM', 'SIGINT').",
			TextEdit:         textEdit("stop_signal: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "storage_opt",
			Detail:           types.CreateStringPointer("object"),
			Documentation:    "Storage driver options for the container.",
			TextEdit:         textEdit(fmt.Sprintf("storage_opt:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "sysctls",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
			TextEdit:         textEdit(fmt.Sprintf("sysctls:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "tmpfs",
			Detail:           types.CreateStringPointer("array or string"),
			Documentation:    "Either a single string or a list of strings.",
			TextEdit:         textEdit("tmpfs:", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "tty",
			Detail:           types.CreateStringPointer("boolean or string"),
			Documentation:    "Allocate a pseudo-TTY to service container.",
			TextEdit:         textEdit("tty: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "ulimits",
			Detail:           types.CreateStringPointer("object"),
			Documentation:    "Container ulimit options, controlling resource limits for processes inside the container.",
			TextEdit:         textEdit(fmt.Sprintf("ulimits:\n%v      ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "use_api_socket",
			Detail:           types.CreateStringPointer("boolean"),
			Documentation:    "Bind mount Docker API socket and required auth.",
			TextEdit:         textEdit("use_api_socket: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "user",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Username or UID to run the container process as.",
			TextEdit:         textEdit("user: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "userns_mode",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "User namespace to use. 'host' shares the host's user namespace.",
			TextEdit:         textEdit("userns_mode: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "uts",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "UTS namespace to use. 'host' shares the host's UTS namespace.",
			TextEdit:         textEdit("uts: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "volumes",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Mount host paths or named volumes accessible to the container. Short syntax (VOLUME:CONTAINER_PATH[:MODE])",
			TextEdit:         textEdit(fmt.Sprintf("volumes:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "volumes_from",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Mount volumes from another service or container. Optionally specify read-only access (ro) or read-write (rw).",
			TextEdit:         textEdit(fmt.Sprintf("volumes_from:\n%v      - ", spacing), line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "working_dir",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "The working directory in which the entrypoint or command will be run",
			TextEdit:         textEdit("working_dir: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
	}
}

func serviceBuildProperties(line, character, prefixLength protocol.UInteger) []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{
			Label:            "additional_contexts",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
			TextEdit:         textEdit("additional_contexts:\n        ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "args",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
			TextEdit:         textEdit("args:\n        ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cache_from",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "List of sources the image builder should use for cache resolution",
			TextEdit:         textEdit("cache_from:\n        - ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "cache_to",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Cache destinations for the build cache.",
			TextEdit:         textEdit("cache_to:\n        - ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "context",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Path to the build context. Can be a relative path or a URL.",
			TextEdit:         textEdit("context: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "dockerfile",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Name of the Dockerfile to use for building the image.",
			TextEdit:         textEdit("dockerfile: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "dockerfile_inline",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Inline Dockerfile content to use instead of a Dockerfile from the build context.",
			TextEdit:         textEdit("dockerfile_inline: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "entitlements",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "List of extra privileged entitlements to grant to the build process.",
			TextEdit:         textEdit("entitlements:\n        - ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "extra_hosts",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "Additional hostnames to be defined in the container's /etc/hosts file.",
			TextEdit:         textEdit("extra_hosts:\n        ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "isolation",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Container isolation technology to use for the build process.",
			TextEdit:         textEdit("isolation: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "labels",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
			TextEdit:         textEdit("labels:\n        ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "network",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Network mode to use for the build. Options include 'default', 'none', 'host', or a network name.",
			TextEdit:         textEdit("network: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "no_cache",
			Detail:           types.CreateStringPointer("boolean or string"),
			Documentation:    "Do not use cache when building the image.",
			TextEdit:         textEdit("no_cache: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "platforms",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Platforms to build for, e.g., 'linux/amd64', 'linux/arm64', or 'windows/amd64'.",
			TextEdit:         textEdit("platforms:\n        - ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "privileged",
			Detail:           types.CreateStringPointer("boolean or string"),
			Documentation:    "Give extended privileges to the build container.",
			TextEdit:         textEdit("privileged: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "provenance",
			Detail:           types.CreateStringPointer("boolean or string"),
			Documentation:    "Add a provenance attestation",
			TextEdit:         textEdit("provenance: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "pull",
			Detail:           types.CreateStringPointer("boolean or string"),
			Documentation:    "Always attempt to pull a newer version of the image.",
			TextEdit:         textEdit("pull: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "sbom",
			Detail:           types.CreateStringPointer("boolean or string"),
			Documentation:    "Add a SBOM attestation",
			TextEdit:         textEdit("sbom: ${1|true,false|}", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "secrets",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Configuration for service configs or secrets, defining how they are mounted in the container.",
			TextEdit:         textEdit("secrets:\n        - ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "shm_size",
			Detail:           types.CreateStringPointer("integer or string"),
			Documentation:    "Size of /dev/shm for the build container. A string value can use suffix like '2g' for 2 gigabytes.",
			TextEdit:         textEdit("shm_size: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "ssh",
			Detail:           types.CreateStringPointer("array or object"),
			Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
			TextEdit:         textEdit("ssh:\n        ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "tags",
			Detail:           types.CreateStringPointer("array"),
			Documentation:    "Additional tags to apply to the built image.",
			TextEdit:         textEdit("tags:\n        - ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "target",
			Detail:           types.CreateStringPointer("string"),
			Documentation:    "Build stage to target in a multi-stage Dockerfile.",
			TextEdit:         textEdit("target: ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
		{
			Label:            "ulimits",
			Detail:           types.CreateStringPointer("object"),
			Documentation:    "Container ulimit options, controlling resource limits for processes inside the container.",
			TextEdit:         textEdit("ulimits:\n        ", line, character, prefixLength),
			InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
			InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
		},
	}
}

func TestCompletion_Schema(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		line      uint32
		character uint32
		list      *protocol.CompletionList
	}{
		{
			name: "top level node suggestions",
			content: `
configs:
  test:
`,
			line:      3,
			character: 0,
			list: &protocol.CompletionList{
				Items: topLevelNodes,
			},
		},
		{
			name:      "top level node suggestions with a space in the front",
			content:   ` `,
			line:      0,
			character: 1,
			list: &protocol.CompletionList{
				Items: topLevelNodes,
			},
		},
		{
			name: "top level node suggestions with indented content but code completion is unindented",
			content: `
 configs:
   test:
`,
			line:      3,
			character: 0,
			list:      nil,
		},
		{
			name: "top level node suggestions with indented content and code completion is aligned correctly",
			content: `
 configs:
   test:
 `,
			line:      3,
			character: 1,
			list: &protocol.CompletionList{
				Items: topLevelNodes,
			},
		},
		{
			name: "alignment correct with multiple documents",
			content: `
---
---
 configs:
   test:
 `,
			line:      5,
			character: 1,
			list: &protocol.CompletionList{
				Items: topLevelNodes,
			},
		},
		{
			name: "alignment incorrect with multiple documents",
			content: `
---
configs:
  test:
---
 configs:
   test2:
`,
			line:      7,
			character: 0,
			list:      nil,
		},
		{
			name: "top level node suggestions with indented content and code completion is aligned correctly but in a comment",
			content: `
 configs:
   test:
#`,
			line:      3,
			character: 1,
			list:      nil,
		},
		{
			name: "top level node suggestions with multiple files",
			content: `
---
 configs:
   test:
---
`,
			line:      5,
			character: 0,
			list: &protocol.CompletionList{
				Items: topLevelNodes,
			},
		},
		{
			name: "comment prevents suggestions",
			content: `
configs:
  test:
#`,
			line:      3,
			character: 1,
			list:      nil,
		},
		{
			name: "config attributes",
			content: `
configs:
  test:
    `,
			line:      3,
			character: 4,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "content",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Inline content of the config.",
						TextEdit:         textEdit("content: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "environment",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Name of an environment variable from which to get the config value.",
						TextEdit:         textEdit("environment: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "external",
						Detail:           types.CreateStringPointer("boolean or object or string"),
						Documentation:    "Specifies that this config already exists and was created outside of Compose.",
						TextEdit:         textEdit("external:", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "file",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path to a file containing the config value.",
						TextEdit:         textEdit("file: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "labels",
						Detail:           types.CreateStringPointer("array or object"),
						Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
						TextEdit:         textEdit("labels:\n      ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "name",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Custom name for this config.",
						TextEdit:         textEdit("name: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "template_driver",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Driver to use for templating the config's value.",
						TextEdit:         textEdit("template_driver: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "network attributes",
			content: `
networks:
  test:
    `,
			line:      3,
			character: 4,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "attachable",
						Detail:           types.CreateStringPointer("boolean or string"),
						Documentation:    "If true, standalone containers can attach to this network.",
						TextEdit:         textEdit("attachable: ${1|true,false|}", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "driver",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Specify which driver should be used for this network. Default is 'bridge'.",
						TextEdit:         textEdit("driver: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "driver_opts",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Specify driver-specific options defined as key/value pairs.",
						TextEdit:         textEdit("driver_opts:\n      ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "enable_ipv4",
						Detail:           types.CreateStringPointer("boolean or string"),
						Documentation:    "Enable IPv4 networking.",
						TextEdit:         textEdit("enable_ipv4: ${1|true,false|}", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "enable_ipv6",
						Detail:           types.CreateStringPointer("boolean or string"),
						Documentation:    "Enable IPv6 networking.",
						TextEdit:         textEdit("enable_ipv6: ${1|true,false|}", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "external",
						Detail:           types.CreateStringPointer("boolean or object or string"),
						Documentation:    "Specifies that this network already exists and was created outside of Compose.",
						TextEdit:         textEdit("external:", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "internal",
						Detail:           types.CreateStringPointer("boolean or string"),
						Documentation:    "Create an externally isolated network.",
						TextEdit:         textEdit("internal: ${1|true,false|}", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "ipam",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Custom IP Address Management configuration for this network.",
						TextEdit:         textEdit("ipam:\n      ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "labels",
						Detail:           types.CreateStringPointer("array or object"),
						Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
						TextEdit:         textEdit("labels:\n      ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "name",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Custom name for this network.",
						TextEdit:         textEdit("name: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "secret attributes",
			content: `
secrets:
  test:
    `,
			line:      3,
			character: 4,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "driver",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Specify which secret driver should be used for this secret.",
						TextEdit:         textEdit("driver: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "driver_opts",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Specify driver-specific options.",
						TextEdit:         textEdit("driver_opts:\n      ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "environment",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Name of an environment variable from which to get the secret value.",
						TextEdit:         textEdit("environment: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "external",
						Detail:           types.CreateStringPointer("boolean or object or string"),
						Documentation:    "Specifies that this secret already exists and was created outside of Compose.",
						TextEdit:         textEdit("external:", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "file",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path to a file containing the secret value.",
						TextEdit:         textEdit("file: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "labels",
						Detail:           types.CreateStringPointer("array or object"),
						Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
						TextEdit:         textEdit("labels:\n      ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "name",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Custom name for this secret.",
						TextEdit:         textEdit("name: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "template_driver",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Driver to use for templating the secret's value.",
						TextEdit:         textEdit("template_driver: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "service attributes",
			content: `
services:
  test:
    `,
			line:      3,
			character: 4,
			list: &protocol.CompletionList{
				Items: serviceProperties(3, 4, 0, ""),
			},
		},
		{
			name: "service attributes respects spacing",
			content: `
  services:
    test:
      `,
			line:      3,
			character: 6,
			list: &protocol.CompletionList{
				Items: serviceProperties(3, 6, 0, "  "),
			},
		},
		{
			name: "prefix of service attributes",
			content: `
services:
  test:
    a`,
			line:      3,
			character: 5,
			list: &protocol.CompletionList{
				Items: serviceProperties(3, 5, 1, ""),
			},
		},
		{
			name: "volume attributes",
			content: `
volumes:
  vol:
    `,
			line:      3,
			character: 4,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "driver",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Specify which volume driver should be used for this volume.",
						TextEdit:         textEdit("driver: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "driver_opts",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Specify driver-specific options.",
						TextEdit:         textEdit("driver_opts:\n      ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "external",
						Detail:           types.CreateStringPointer("boolean or object or string"),
						Documentation:    "Specifies that this volume already exists and was created outside of Compose.",
						TextEdit:         textEdit("external:", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "labels",
						Detail:           types.CreateStringPointer("array or object"),
						Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
						TextEdit:         textEdit("labels:\n      ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "name",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Custom name for this volume.",
						TextEdit:         textEdit("name: ", 3, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "top level attributes show nothing",
			content: `
configs:
volumes: `,
			line:      2,
			character: 9,
			list:      nil,
		},
		{
			name: "node suggestions do not bleed over to the next top level node",
			content: `
configs:
  configA:
volumes: `,
			line:      3,
			character: 9,
			list:      nil,
		},
		{
			name: "inner attributes of the blkio_config object under service",
			content: `
services:
  postgres:
    blkio_config:
      `,
			line:      4,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "device_read_bps",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "Limit read rate (bytes per second) from a device.",
						TextEdit:         textEdit("device_read_bps:\n        - ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "device_read_iops",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "Limit read rate (IO per second) from a device.",
						TextEdit:         textEdit("device_read_iops:\n        - ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "device_write_bps",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "Limit write rate (bytes per second) to a device.",
						TextEdit:         textEdit("device_write_bps:\n        - ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "device_write_iops",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "Limit write rate (IO per second) to a device.",
						TextEdit:         textEdit("device_write_iops:\n        - ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "weight",
						Detail:           types.CreateStringPointer("integer or string"),
						Documentation:    "Block IO weight (relative weight) for the service, between 10 and 1000.",
						TextEdit:         textEdit("weight: ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "weight_device",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "Block IO weight (relative weight) for specific devices.",
						TextEdit:         textEdit("weight_device:\n        - ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "inner attributes of the deploy object under service",
			content: `
services:
  postgres:
    deploy:
      `,
			line:      4,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "endpoint_mode",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Endpoint mode for the service: 'vip' (default) or 'dnsrr'.",
						TextEdit:         textEdit("endpoint_mode: ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "labels",
						Detail:           types.CreateStringPointer("array or object"),
						Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
						TextEdit:         textEdit("labels:\n        ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "mode",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Deployment mode for the service: 'replicated' (default) or 'global'.",
						TextEdit:         textEdit("mode: ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "placement",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Constraints and preferences for the platform to select a physical node to run service containers",
						TextEdit:         textEdit("placement:\n        ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "replicas",
						Detail:           types.CreateStringPointer("integer or string"),
						Documentation:    "Number of replicas of the service container to run.",
						TextEdit:         textEdit("replicas: ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "resources",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Resource constraints and reservations for the service.",
						TextEdit:         textEdit("resources:\n        ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "restart_policy",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Restart policy for the service containers.",
						TextEdit:         textEdit("restart_policy:\n        ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "rollback_config",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration for rolling back a service update.",
						TextEdit:         textEdit("rollback_config:\n        ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "update_config",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration for updating a service.",
						TextEdit:         textEdit("update_config:\n        ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "attributes of the develop's watch array items",
			content: `
services:
  postgres:
    develop:
      watch:
        - `,
			line:      5,
			character: 10,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "action",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Action to take when a change is detected: rebuild the container, sync files, restart the container, sync and restart, or sync and execute a command.",
						TextEdit:         textEdit("action: ${1|rebuild,restart,sync,sync+exec,sync+restart|}", 5, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "exec",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration for service lifecycle hooks, which are commands executed at specific points in a container's lifecycle.",
						TextEdit:         textEdit("exec:\n            ", 5, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "ignore",
						Detail:           types.CreateStringPointer("array or string"),
						Documentation:    "Either a single string or a list of strings.",
						TextEdit:         textEdit("ignore:", 5, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "include",
						Detail:           types.CreateStringPointer("array or string"),
						Documentation:    "Either a single string or a list of strings.",
						TextEdit:         textEdit("include:", 5, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "initial_sync",
						Detail:           types.CreateStringPointer("boolean"),
						Documentation:    "Ensure that an initial synchronization is done before starting watch mode for sync+x triggers",
						TextEdit:         textEdit("initial_sync: ${1|true,false|}", 5, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "path",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path to watch for changes.",
						TextEdit:         textEdit("path: ", 5, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Target path in the container for sync operations.",
						TextEdit:         textEdit("target: ", 5, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "inner attributes of the deploy/resources object under service",
			content: `
services:
  postgres:
    deploy:
      resources:
        `,
			line:      5,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "limits",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Resource limits for the service containers.",
						TextEdit:         textEdit("limits:\n          ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "reservations",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Resource reservations for the service containers.",
						TextEdit:         textEdit("reservations:\n          ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "inner attributes of the develop object under service",
			content: `
services:
  postgres:
    develop:
      `,
			line:      4,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "watch",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "Configure watch mode for the service, which monitors file changes and performs actions in response.",
						TextEdit:         textEdit("watch:\n        - action: ${1}\n          path: ${2}", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "inner attributes of a model object under the service",
			content: `
services:
  llm:
    models:
      other:
        `,
			line:      5,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "endpoint_var",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Environment variable set to AI model endpoint.",
						TextEdit:         textEdit("endpoint_var: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "model_var",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Environment variable set to AI model name.",
						TextEdit:         textEdit("model_var: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "inner attributes of the build object under service",
			content: `
services:
  postgres:
    build:
      `,
			line:      4,
			character: 6,
			list: &protocol.CompletionList{
				Items: serviceBuildProperties(4, 6, 0),
			},
		},
		{
			name: "inner attributes of the blkio_config object under service with weight already present",
			content: `
services:
  postgres:
    blkio_config:
      weight: 0
      `,
			line:      5,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "device_read_bps",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "Limit read rate (bytes per second) from a device.",
						TextEdit:         textEdit("device_read_bps:\n        - ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "device_read_iops",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "Limit read rate (IO per second) from a device.",
						TextEdit:         textEdit("device_read_iops:\n        - ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "device_write_bps",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "Limit write rate (bytes per second) to a device.",
						TextEdit:         textEdit("device_write_bps:\n        - ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "device_write_iops",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "Limit write rate (IO per second) to a device.",
						TextEdit:         textEdit("device_write_iops:\n        - ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "weight",
						Detail:           types.CreateStringPointer("integer or string"),
						Documentation:    "Block IO weight (relative weight) for the service, between 10 and 1000.",
						TextEdit:         textEdit("weight: ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "weight_device",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "Block IO weight (relative weight) for specific devices.",
						TextEdit:         textEdit("weight_device:\n        - ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "inner attributes of the configs array object under service",
			content: `
services:
  test:
    configs:
    - `,
			line:      4,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "gid",
						Detail:           types.CreateStringPointer("string"),
						TextEdit:         textEdit("gid: ", 4, 6, 0),
						Documentation:    "GID of the file in the container. Default is 0 (root).",
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "mode",
						Detail:           types.CreateStringPointer("number or string"),
						Documentation:    "File permission mode inside the container, in octal. Default is 0444 for configs and 0400 for secrets.",
						TextEdit:         textEdit("mode: ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Name of the config or secret as defined in the top-level configs or secrets section.",
						TextEdit:         textEdit("source: ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						TextEdit:         textEdit("target: ", 4, 6, 0),
						Documentation:    "Path in the container where the config or secret will be mounted. Defaults to /<source> for configs and /run/secrets/<source> for secrets.",
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "uid",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "UID of the file in the container. Default is 0 (root).",
						TextEdit:         textEdit("uid: ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "patternProperties triggered which should return nothing",
			content: `
services:
  test:
    build:
      ulimits:
        `,
			line:      5,
			character: 8,
			list:      nil,
		},
		{
			name: "inner attributes of the ulimits object under service",
			content: `
services:
  test:
    build:
      ulimits:
        abc:
          `,
			line:      6,
			character: 10,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "hard",
						Detail:           types.CreateStringPointer("integer or string"),
						Documentation:    "Hard limit for the ulimit type. This is the maximum allowed value.",
						TextEdit:         textEdit("hard: ", 6, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "soft",
						Detail:           types.CreateStringPointer("integer or string"),
						Documentation:    "Soft limit for the ulimit type. This is the value that's actually enforced.",
						TextEdit:         textEdit("soft: ", 6, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "indentation considered for a sibling of a service's attribute",
			content: `
services:
  postgres:
    image: alpine
    `,
			line:      4,
			character: 4,
			list: &protocol.CompletionList{
				Items: serviceProperties(4, 4, 0, ""),
			},
		},
		{
			name: "indentation considered and does not suggest any items needing a name",
			content: `
services:
  postgres:
    blkio_config:
      weight: 0
  `,
			line:      5,
			character: 2,
			list:      nil,
		},
		{
			name: "indentation considered when suggesting child items",
			content: `
services:
  postgres:
    blkio_config:
      weight: 0
    `,
			line:      5,
			character: 4,
			list: &protocol.CompletionList{
				Items: serviceProperties(5, 4, 0, ""),
			},
		},
		{
			name: "invalid services as an array",
			content: `
services:
  - `,
			line:      2,
			character: 4,
			list:      nil,
		},
		{
			name: "properties of an embedded object with a custom name",
			content: `
services:
  test:
    networks:
      abc:
        `,
			line:      5,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "aliases",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "A list of unique string values.",
						TextEdit:         textEdit("aliases:\n          - ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "driver_opts",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Driver options for this network.",
						TextEdit:         textEdit("driver_opts:\n          ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "gw_priority",
						Detail:           types.CreateStringPointer("number"),
						Documentation:    "Specify the gateway priority for the network connection.",
						TextEdit:         textEdit("gw_priority: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "interface_name",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Interface network name used to connect to network",
						TextEdit:         textEdit("interface_name: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "ipv4_address",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Specify a static IPv4 address for this service on this network.",
						TextEdit:         textEdit("ipv4_address: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "ipv6_address",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Specify a static IPv6 address for this service on this network.",
						TextEdit:         textEdit("ipv6_address: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "link_local_ips",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "A list of unique string values.",
						TextEdit:         textEdit("link_local_ips:\n          - ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "mac_address",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Specify a MAC address for this service on this network.",
						TextEdit:         textEdit("mac_address: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "priority",
						Detail:           types.CreateStringPointer("number"),
						Documentation:    "Specify the priority for the network connection.",
						TextEdit:         textEdit("priority: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "properties of an embedded object with a custom name with extra trailing whitespace",
			content: `
services:
  test:
    networks:
      abc:
          `,
			line:      5,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "aliases",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "A list of unique string values.",
						TextEdit:         textEdit("aliases:\n          - ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "driver_opts",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Driver options for this network.",
						TextEdit:         textEdit("driver_opts:\n          ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "gw_priority",
						Detail:           types.CreateStringPointer("number"),
						Documentation:    "Specify the gateway priority for the network connection.",
						TextEdit:         textEdit("gw_priority: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "interface_name",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Interface network name used to connect to network",
						TextEdit:         textEdit("interface_name: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "ipv4_address",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Specify a static IPv4 address for this service on this network.",
						TextEdit:         textEdit("ipv4_address: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "ipv6_address",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Specify a static IPv6 address for this service on this network.",
						TextEdit:         textEdit("ipv6_address: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "link_local_ips",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "A list of unique string values.",
						TextEdit:         textEdit("link_local_ips:\n          - ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "mac_address",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Specify a MAC address for this service on this network.",
						TextEdit:         textEdit("mac_address: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "priority",
						Detail:           types.CreateStringPointer("number"),
						Documentation:    "Specify the priority for the network connection.",
						TextEdit:         textEdit("priority: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "oneOf results of a service object's networks attribute",
			content: `
services:
  test:
    networks:
      `,
			line:      4,
			character: 6,
			list:      nil,
		},
		{
			name: "sibling attributes shown after an array of items",
			content: `
services:
  test:
    networks:
    - testNetwork
    `,
			line:      5,
			character: 4,
			list: &protocol.CompletionList{
				Items: serviceProperties(5, 4, 0, ""),
			},
		},
		{
			name: "properties of a volumes array item with no content",
			content: `
services:
  test:
    image: alpine
    volumes:
      - `,
			line:      5,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "bind",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to bind mounts.",
						TextEdit:         textEdit("bind:\n          ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "consistency",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The consistency requirements for the mount. Available values are platform specific.",
						TextEdit:         textEdit("consistency: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "image",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to image mounts.",
						TextEdit:         textEdit("image:\n          ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "read_only",
						Detail:           types.CreateStringPointer("boolean or string"),
						Documentation:    "Flag to set the volume as read-only.",
						TextEdit:         textEdit("read_only: ${1|true,false|}", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The source of the mount, a path on the host for a bind mount, a docker image reference for an image mount, or the name of a volume defined in the top-level volumes key. Not applicable for a tmpfs mount.",
						TextEdit:         textEdit("source: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The path in the container where the volume is mounted.",
						TextEdit:         textEdit("target: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "tmpfs",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to tmpfs mounts.",
						TextEdit:         textEdit("tmpfs:\n          ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "type",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						TextEdit:         textEdit("type: ${1|bind,cluster,image,npipe,tmpfs,volume|}", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "volume",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to volume mounts.",
						TextEdit:         textEdit("volume:\n          ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "properties of a volume array item's sibling attributes under a service object",
			content: `
services:
  test:
    image: alpine
    volumes:
      - type: bind
        `,
			line:      6,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "bind",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to bind mounts.",
						TextEdit:         textEdit("bind:\n          ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "consistency",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The consistency requirements for the mount. Available values are platform specific.",
						TextEdit:         textEdit("consistency: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "image",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to image mounts.",
						TextEdit:         textEdit("image:\n          ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "read_only",
						Detail:           types.CreateStringPointer("boolean or string"),
						Documentation:    "Flag to set the volume as read-only.",
						TextEdit:         textEdit("read_only: ${1|true,false|}", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The source of the mount, a path on the host for a bind mount, a docker image reference for an image mount, or the name of a volume defined in the top-level volumes key. Not applicable for a tmpfs mount.",
						TextEdit:         textEdit("source: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The path in the container where the volume is mounted.",
						TextEdit:         textEdit("target: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "tmpfs",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to tmpfs mounts.",
						TextEdit:         textEdit("tmpfs:\n          ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "type",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						TextEdit:         textEdit("type: ${1|bind,cluster,image,npipe,tmpfs,volume|}", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "volume",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to volume mounts.",
						TextEdit:         textEdit("volume:\n          ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "properties of a volume array item's bind attributes under a service object",
			content: `
services:
  test:
    image: alpine
    volumes:
      - type: bind
        bind:
          `,
			line:      7,
			character: 10,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "create_host_path",
						Detail:           types.CreateStringPointer("boolean or string"),
						Documentation:    "Create the host path if it doesn't exist.",
						TextEdit:         textEdit("create_host_path: ${1|true,false|}", 7, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "propagation",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The propagation mode for the bind mount: 'shared', 'slave', 'private', 'rshared', 'rslave', or 'rprivate'.",
						TextEdit:         textEdit("propagation: ", 7, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "recursive",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Recursively mount the source directory.",
						TextEdit:         textEdit("recursive: ${1|disabled,enabled,readonly,writable|}", 7, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "selinux",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "SELinux relabeling options: 'z' for shared content, 'Z' for private unshared content.",
						TextEdit:         textEdit("selinux: ${1|Z,z|}", 7, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "properties of a volume array item's volume attributes under a service object",
			content: `
services:
  test:
    image: alpine
    volumes:
      - type: bind
        volume:
          `,
			line:      7,
			character: 10,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "labels",
						Detail:           types.CreateStringPointer("array or object"),
						Documentation:    "Either a dictionary mapping keys to values, or a list of strings.",
						TextEdit:         textEdit("labels:\n            ", 7, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "nocopy",
						Detail:           types.CreateStringPointer("boolean or string"),
						Documentation:    "Flag to disable copying of data from a container when a volume is created.",
						TextEdit:         textEdit("nocopy: ${1|true,false|}", 7, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "subpath",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path within the volume to mount instead of the volume root.",
						TextEdit:         textEdit("subpath: ", 7, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "enum properties for a string attribute directly as an item is suggested",
			content: `
services:
  test:
    volumes:
      - type: `,
			line:      4,
			character: 14,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:         "bind",
						Documentation: "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("bind", 4, 14, 0),
					},
					{
						Label:         "cluster",
						Documentation: "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("cluster", 4, 14, 0),
					},
					{
						Label:         "image",
						Documentation: "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("image", 4, 14, 0),
					},
					{
						Label:         "npipe",
						Documentation: "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("npipe", 4, 14, 0),
					},
					{
						Label:         "tmpfs",
						Documentation: "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("tmpfs", 4, 14, 0),
					},
					{
						Label:         "volume",
						Documentation: "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("volume", 4, 14, 0),
					},
				},
			},
		},
		{
			name: "enum properties for a string attribute considers the prefix for the TextEdit",
			content: `
services:
  test:
    volumes:
      - type: b`,
			line:      4,
			character: 15,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:         "bind",
						Documentation: "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("bind", 4, 15, 1),
					},
					{
						Label:         "cluster",
						Documentation: "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("cluster", 4, 15, 1),
					},
					{
						Label:         "image",
						Documentation: "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("image", 4, 15, 1),
					},
					{
						Label:         "npipe",
						Documentation: "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("npipe", 4, 15, 1),
					},
					{
						Label:         "tmpfs",
						Documentation: "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("tmpfs", 4, 15, 1),
					},
					{
						Label:         "volume",
						Documentation: "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("volume", 4, 15, 1),
					},
				},
			},
		},
		{
			name: "enum properties for a string attribute is suggested",
			content: `
services:
  test:
    volumes:
      - type: bind
        bind:
          selinux: `,
			line:      6,
			character: 19,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:         "Z",
						Documentation: "SELinux relabeling options: 'z' for shared content, 'Z' for private unshared content.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("Z", 6, 19, 0),
					},
					{
						Label:         "z",
						Documentation: "SELinux relabeling options: 'z' for shared content, 'Z' for private unshared content.",
						Detail:        types.CreateStringPointer("string"),
						TextEdit:      textEdit("z", 6, 19, 0),
					},
				},
			},
		},
		{
			name: "nothing suggested for an arbitrary string attribute's value",
			content: `
services:
  test:
    volumes:
      - type: bind
        bind:
          propagation:`,
			line:      6,
			character: 22,
			list:      nil,
		},
		{
			name: "unexpected array for networks",
			content: `
networks:
- 
- 
- `,
			line:      4,
			character: 2,
			list:      nil,
		},
		{
			name: "configs attributes creates an array if necessary",
			content: `
services:
  test:
    configs:
      `,
			line:      4,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "gid",
						Detail:           types.CreateStringPointer("string"),
						TextEdit:         textEdit("- gid: ", 4, 6, 0),
						Documentation:    "GID of the file in the container. Default is 0 (root).",
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "mode",
						Detail:           types.CreateStringPointer("number or string"),
						Documentation:    "File permission mode inside the container, in octal. Default is 0444 for configs and 0400 for secrets.",
						TextEdit:         textEdit("- mode: ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Name of the config or secret as defined in the top-level configs or secrets section.",
						TextEdit:         textEdit("- source: ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path in the container where the config or secret will be mounted. Defaults to /<source> for configs and /run/secrets/<source> for secrets.",
						TextEdit:         textEdit("- target: ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "uid",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "UID of the file in the container. Default is 0 (root).",
						TextEdit:         textEdit("- uid: ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "array items should still consider indentation when calculating completion items",
			content: `
services:
  test:
    configs:
      - mode: 0
      `,
			line:      5,
			character: 6,
			list:      nil,
		},
		{
			name: "indentation considered for regexp object properties",
			content: `
services:
  test:
    networks:
      abc:
      `,
			line:      5,
			character: 6,
			list:      nil,
		},
		{
			name: "reservations completion",
			content: `
services:
  test:
    image: redis
    deploy:
      resources:
        reservations:
          `,
			line:      7,
			character: 10,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "cpus",
						Detail:           types.CreateStringPointer("number or string"),
						Documentation:    "Reservation for how much of the available CPU resources, as number of cores, a container can use.",
						TextEdit:         textEdit("cpus: ", 7, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "devices",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "Device reservations for containers, allowing services to access specific hardware devices.",
						TextEdit:         textEdit("devices:\n            - capabilities:\n              - ", 7, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "generic_resources",
						Detail:           types.CreateStringPointer("array"),
						Documentation:    "User-defined resources for services, allowing services to reserve specialized hardware resources.",
						TextEdit:         textEdit("generic_resources:\n            - ", 7, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "memory",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Reservation on the amount of memory a container can allocate (e.g., '1g', '1024m').",
						TextEdit:         textEdit("memory: ", 7, 10, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "param character is outside document range",
			content: `
services:
  test:
    `,
			line:      3,
			character: 5,
			list:      nil,
		},
		{
			name:      "param line is outside document range",
			content:   "",
			line:      1,
			character: 0,
			list:      nil,
		},
		{
			name:      "completion is on a line before the first node and after it positionally",
			content:   " \nservices:",
			line:      0,
			character: 1,
			list:      nil,
		},
		{
			name:      "completion is on a line before the first node and before it positionally",
			content:   " \n  services:",
			line:      0,
			character: 1,
			list:      nil,
		},
		{
			name:      "include properties with no array hyphen present",
			content:   "include:\n  ",
			line:      1,
			character: 2,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "env_file",
						Detail:           types.CreateStringPointer("array or string"),
						Documentation:    "Either a single string or a list of strings.",
						TextEdit:         textEdit("- env_file:", 1, 2, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "path",
						Detail:           types.CreateStringPointer("array or string"),
						Documentation:    "Either a single string or a list of strings.",
						TextEdit:         textEdit("- path:", 1, 2, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "project_directory",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path to resolve relative paths set in the Compose file",
						TextEdit:         textEdit("- project_directory: ", 1, 2, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name:      "include properties with array hyphen present",
			content:   "include:\n  - ",
			line:      1,
			character: 4,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "env_file",
						Detail:           types.CreateStringPointer("array or string"),
						Documentation:    "Either a single string or a list of strings.",
						TextEdit:         textEdit("env_file:", 1, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "path",
						Detail:           types.CreateStringPointer("array or string"),
						Documentation:    "Either a single string or a list of strings.",
						TextEdit:         textEdit("path:", 1, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "project_directory",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path to resolve relative paths set in the Compose file",
						TextEdit:         textEdit("project_directory: ", 1, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name:      "include properties with a prefix",
			content:   "include:\n  - e",
			line:      1,
			character: 5,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "env_file",
						Detail:           types.CreateStringPointer("array or string"),
						Documentation:    "Either a single string or a list of strings.",
						TextEdit:         textEdit("env_file:", 1, 5, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "path",
						Detail:           types.CreateStringPointer("array or string"),
						Documentation:    "Either a single string or a list of strings.",
						TextEdit:         textEdit("path:", 1, 5, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "project_directory",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path to resolve relative paths set in the Compose file",
						TextEdit:         textEdit("project_directory: ", 1, 5, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name:      "empty JSON object",
			content:   "{}",
			line:      0,
			character: 0,
			list:      nil,
		},
	}

	dir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("%v-%v", t.Name(), time.Now().UnixMilli()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(dir))
	})
	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(dir, "compose.yaml")), "/"))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			hub := hub.NewService()
			doc := document.NewComposeDocument(manager, uri.URI(composeFileURI), 1, []byte(tc.content))
			list, err := Completion(context.Background(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: composeFileURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, manager, &hub, doc)
			require.NoError(t, err)
			require.Equal(t, tc.list, list)
		})
	}
}

func TestCompletion_NamedDependencies(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		line      uint32
		character uint32
		list      *protocol.CompletionList
	}{
		{
			name: "depends_on array items",
			content: `
services:
  test:
    image: alpine
    depends_on:
      - 
  test2:
    image: alpine`,
			line:      5,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 5, 8, 0),
					},
				},
			},
		},
		{
			name: "depends_on array items across two files",
			content: `
---
services:
  test:
    image: alpine
    depends_on:
      - 
---
services:
  test2:
    image: alpine`,
			line:      6,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 6, 8, 0),
					},
				},
			},
		},
		{
			name: "depends_on array items with a prefix",
			content: `
services:
  test:
    image: alpine
    depends_on:
      - t
  test2:
    image: alpine`,
			line:      5,
			character: 9,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 5, 9, 1),
					},
				},
			},
		},
		{
			name: "depends_on service object",
			content: `
services:
  test:
    image: alpine
    depends_on:
      
  test2:
    image: alpine`,
			line:      5,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 5, 6, 0),
					},
				},
			},
		},
		{
			name: "extends as a string",
			content: `
services:
  test:
    image: alpine
    extends: 
  test2:
    image: alpine`,
			line:      4,
			character: 13,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 4, 13, 0),
					},
				},
			},
		},
		{
			name: "extends as an object with the service attribute",
			content: `
services:
  test:
    image: alpine
    extends:
      service: 
  test2:
    image: alpine`,
			line:      5,
			character: 15,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 5, 15, 0),
					},
				},
			},
		},
		{
			name: "extends object attributes with no other services found",
			content: `
services:
  test:
    image: alpine
    extends:
      `,
			line:      5,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "file",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The file path where the service to extend is defined.",
						TextEdit:         textEdit("file: ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "service",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The name of the service to extend.",
						TextEdit:         textEdit("service: ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "extends object attributes with a service defined",
			content: `
services:
  test:
    image: alpine
    extends:
      
  test2:
    image: alpine`,
			line:      5,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "file",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The file path where the service to extend is defined.",
						TextEdit:         textEdit("file: ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "service",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The name of the service to extend.",
						TextEdit:         textEdit("service: ${1|test2|}", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "extends object attributes with a file pointing at itself",
			content: `
services:
  test:
    image: alpine
    extends:
      file: compose.yaml
      
  test2:
    image: alpine`,
			line:      6,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "file",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The file path where the service to extend is defined.",
						TextEdit:         textEdit("file: ", 6, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "service",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The name of the service to extend.",
						TextEdit:         textEdit("service: ${1|test2|}", 6, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "extends object attributes with a file pointing somewhere else",
			content: `
services:
  test:
    image: alpine
    extends:
      file: non-existent-compose.yaml
      
  test2:
    image: alpine`,
			line:      6,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "file",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The file path where the service to extend is defined.",
						TextEdit:         textEdit("file: ", 6, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "service",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The name of the service to extend.",
						TextEdit:         textEdit("service: ", 6, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "extends' service attribute with a file pointing somewhere else",
			content: `
services:
  test:
    image: alpine
    extends:
      file: non-existent-compose.yaml
      service: 
  test2:
    image: alpine`,
			line:      6,
			character: 15,
			list:      nil,
		},
		{
			name: "networks array items",
			content: `
services:
  test:
    image: alpine
    networks:
      - 
networks:
  test2:
    image: alpine`,
			line:      5,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 5, 8, 0),
					},
				},
			},
		},
		{
			name: "networks array items across two files",
			content: `
---
services:
  test:
    image: alpine
    networks:
      - 
---
networks:
  test2:`,
			line:      6,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 6, 8, 0),
					},
				},
			},
		},
		{
			name: "networks array items with a prefix",
			content: `
services:
  test:
    image: alpine
    networks:
      - t
networks:
  test2:`,
			line:      5,
			character: 9,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 5, 9, 1),
					},
				},
			},
		},
		{
			name: "networks service object",
			content: `
services:
  test:
    image: alpine
    networks:
      
networks:
  test2:`,
			line:      5,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 5, 6, 0),
					},
				},
			},
		},
		{
			name: "volumes array item shows item properties and named references",
			content: `
services:
  test:
    image: alpine
    volumes:
      - 
volumes:
  test2:`,
			line:      5,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "bind",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to bind mounts.",
						TextEdit:         textEdit("bind:\n          ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "consistency",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The consistency requirements for the mount. Available values are platform specific.",
						TextEdit:         textEdit("consistency: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "image",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to image mounts.",
						TextEdit:         textEdit("image:\n          ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "read_only",
						Detail:           types.CreateStringPointer("boolean or string"),
						Documentation:    "Flag to set the volume as read-only.",
						TextEdit:         textEdit("read_only: ${1|true,false|}", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The source of the mount, a path on the host for a bind mount, a docker image reference for an image mount, or the name of a volume defined in the top-level volumes key. Not applicable for a tmpfs mount.",
						TextEdit:         textEdit("source: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The path in the container where the volume is mounted.",
						TextEdit:         textEdit("target: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "test2",
						TextEdit:         textEdit("test2:${1:/container/path}", 5, 8, 0),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "tmpfs",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to tmpfs mounts.",
						TextEdit:         textEdit("tmpfs:\n          ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "type",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						TextEdit:         textEdit("type: ${1|bind,cluster,image,npipe,tmpfs,volume|}", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "volume",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to volume mounts.",
						TextEdit:         textEdit("volume:\n          ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "volumes array items across two YAML documents in a single file",
			content: `
---
services:
  test:
    image: alpine
    volumes:
      - 
---
volumes:
  test2:`,
			line:      6,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "bind",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to bind mounts.",
						TextEdit:         textEdit("bind:\n          ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "consistency",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The consistency requirements for the mount. Available values are platform specific.",
						TextEdit:         textEdit("consistency: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "image",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to image mounts.",
						TextEdit:         textEdit("image:\n          ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "read_only",
						Detail:           types.CreateStringPointer("boolean or string"),
						Documentation:    "Flag to set the volume as read-only.",
						TextEdit:         textEdit("read_only: ${1|true,false|}", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The source of the mount, a path on the host for a bind mount, a docker image reference for an image mount, or the name of a volume defined in the top-level volumes key. Not applicable for a tmpfs mount.",
						TextEdit:         textEdit("source: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The path in the container where the volume is mounted.",
						TextEdit:         textEdit("target: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "test2",
						TextEdit:         textEdit("test2:${1:/container/path}", 6, 8, 0),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "tmpfs",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to tmpfs mounts.",
						TextEdit:         textEdit("tmpfs:\n          ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "type",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						TextEdit:         textEdit("type: ${1|bind,cluster,image,npipe,tmpfs,volume|}", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "volume",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to volume mounts.",
						TextEdit:         textEdit("volume:\n          ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "volumes array items with a prefix",
			content: `
services:
  test:
    image: alpine
    volumes:
      - t
volumes:
  test2:`,
			line:      5,
			character: 9,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "bind",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to bind mounts.",
						TextEdit:         textEdit("bind:\n          ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "consistency",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The consistency requirements for the mount. Available values are platform specific.",
						TextEdit:         textEdit("consistency: ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "image",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to image mounts.",
						TextEdit:         textEdit("image:\n          ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "read_only",
						Detail:           types.CreateStringPointer("boolean or string"),
						Documentation:    "Flag to set the volume as read-only.",
						TextEdit:         textEdit("read_only: ${1|true,false|}", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The source of the mount, a path on the host for a bind mount, a docker image reference for an image mount, or the name of a volume defined in the top-level volumes key. Not applicable for a tmpfs mount.",
						TextEdit:         textEdit("source: ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The path in the container where the volume is mounted.",
						TextEdit:         textEdit("target: ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "test2",
						TextEdit:         textEdit("test2:${1:/container/path}", 5, 9, 1),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "tmpfs",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to tmpfs mounts.",
						TextEdit:         textEdit("tmpfs:\n          ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "type",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						TextEdit:         textEdit("type: ${1|bind,cluster,image,npipe,tmpfs,volume|}", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "volume",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to volume mounts.",
						TextEdit:         textEdit("volume:\n          ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "volumes service object",
			content: `
services:
  test:
    image: alpine
    volumes:
      
volumes:
  test2:`,
			line:      5,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "bind",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to bind mounts.",
						TextEdit:         textEdit("- bind:\n          ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "consistency",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The consistency requirements for the mount. Available values are platform specific.",
						TextEdit:         textEdit("- consistency: ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "image",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to image mounts.",
						TextEdit:         textEdit("- image:\n          ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "read_only",
						Detail:           types.CreateStringPointer("boolean or string"),
						Documentation:    "Flag to set the volume as read-only.",
						TextEdit:         textEdit("- read_only: ${1|true,false|}", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The source of the mount, a path on the host for a bind mount, a docker image reference for an image mount, or the name of a volume defined in the top-level volumes key. Not applicable for a tmpfs mount.",
						TextEdit:         textEdit("- source: ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The path in the container where the volume is mounted.",
						TextEdit:         textEdit("- target: ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "test2",
						TextEdit:         textEdit("- test2:${1:/container/path}", 5, 6, 0),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "tmpfs",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to tmpfs mounts.",
						TextEdit:         textEdit("- tmpfs:\n          ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "type",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "The mount type: bind for mounting host directories, volume for named volumes, tmpfs for temporary filesystems, cluster for cluster volumes, npipe for named pipes, or image for mounting from an image.",
						TextEdit:         textEdit("- type: ${1|bind,cluster,image,npipe,tmpfs,volume|}", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "volume",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Configuration specific to volume mounts.",
						TextEdit:         textEdit("- volume:\n          ", 5, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "configs array items",
			content: `
services:
  test:
    image: alpine
    configs:
      - 
configs:
  test2:
    file: ./httpd.conf`,
			line:      5,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "gid",
						Detail:           types.CreateStringPointer("string"),
						TextEdit:         textEdit("gid: ", 5, 8, 0),
						Documentation:    "GID of the file in the container. Default is 0 (root).",
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "mode",
						Detail:           types.CreateStringPointer("number or string"),
						Documentation:    "File permission mode inside the container, in octal. Default is 0444 for configs and 0400 for secrets.",
						TextEdit:         textEdit("mode: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Name of the config or secret as defined in the top-level configs or secrets section.",
						TextEdit:         textEdit("source: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path in the container where the config or secret will be mounted. Defaults to /<source> for configs and /run/secrets/<source> for secrets.",
						TextEdit:         textEdit("target: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 5, 8, 0),
					},
					{
						Label:            "uid",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "UID of the file in the container. Default is 0 (root).",
						TextEdit:         textEdit("uid: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "configs array items across two files",
			content: `
---
services:
  test:
    image: alpine
    configs:
      - 
---
configs:
  test2:
    file: ./httpd.conf`,
			line:      6,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "gid",
						Detail:           types.CreateStringPointer("string"),
						TextEdit:         textEdit("gid: ", 6, 8, 0),
						Documentation:    "GID of the file in the container. Default is 0 (root).",
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "mode",
						Detail:           types.CreateStringPointer("number or string"),
						Documentation:    "File permission mode inside the container, in octal. Default is 0444 for configs and 0400 for secrets.",
						TextEdit:         textEdit("mode: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Name of the config or secret as defined in the top-level configs or secrets section.",
						TextEdit:         textEdit("source: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path in the container where the config or secret will be mounted. Defaults to /<source> for configs and /run/secrets/<source> for secrets.",
						TextEdit:         textEdit("target: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 6, 8, 0),
					},
					{
						Label:            "uid",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "UID of the file in the container. Default is 0 (root).",
						TextEdit:         textEdit("uid: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "configs array items with a prefix",
			content: `
services:
  test:
    image: alpine
    configs:
      - t
configs:
  test2:
    file: ./httpd.conf`,
			line:      5,
			character: 9,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "gid",
						Detail:           types.CreateStringPointer("string"),
						TextEdit:         textEdit("gid: ", 5, 9, 1),
						Documentation:    "GID of the file in the container. Default is 0 (root).",
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "mode",
						Detail:           types.CreateStringPointer("number or string"),
						Documentation:    "File permission mode inside the container, in octal. Default is 0444 for configs and 0400 for secrets.",
						TextEdit:         textEdit("mode: ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Name of the config or secret as defined in the top-level configs or secrets section.",
						TextEdit:         textEdit("source: ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path in the container where the config or secret will be mounted. Defaults to /<source> for configs and /run/secrets/<source> for secrets.",
						TextEdit:         textEdit("target: ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 5, 9, 1),
					},
					{
						Label:            "uid",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "UID of the file in the container. Default is 0 (root).",
						TextEdit:         textEdit("uid: ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "secrets array items",
			content: `
services:
  test:
    image: alpine
    secrets:
      - 
secrets:
  test2:
    file: ./httpd.conf`,
			line:      5,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "gid",
						Detail:           types.CreateStringPointer("string"),
						TextEdit:         textEdit("gid: ", 5, 8, 0),
						Documentation:    "GID of the file in the container. Default is 0 (root).",
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "mode",
						Detail:           types.CreateStringPointer("number or string"),
						Documentation:    "File permission mode inside the container, in octal. Default is 0444 for configs and 0400 for secrets.",
						TextEdit:         textEdit("mode: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Name of the config or secret as defined in the top-level configs or secrets section.",
						TextEdit:         textEdit("source: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path in the container where the config or secret will be mounted. Defaults to /<source> for configs and /run/secrets/<source> for secrets.",
						TextEdit:         textEdit("target: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 5, 8, 0),
					},
					{
						Label:            "uid",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "UID of the file in the container. Default is 0 (root).",
						TextEdit:         textEdit("uid: ", 5, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "secrets array items across two files",
			content: `
---
services:
  test:
    image: alpine
    secrets:
      - 
---
secrets:
  test2:
    file: ./httpd.conf`,
			line:      6,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "gid",
						Detail:           types.CreateStringPointer("string"),
						TextEdit:         textEdit("gid: ", 6, 8, 0),
						Documentation:    "GID of the file in the container. Default is 0 (root).",
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "mode",
						Detail:           types.CreateStringPointer("number or string"),
						Documentation:    "File permission mode inside the container, in octal. Default is 0444 for configs and 0400 for secrets.",
						TextEdit:         textEdit("mode: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Name of the config or secret as defined in the top-level configs or secrets section.",
						TextEdit:         textEdit("source: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path in the container where the config or secret will be mounted. Defaults to /<source> for configs and /run/secrets/<source> for secrets.",
						TextEdit:         textEdit("target: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 6, 8, 0),
					},
					{
						Label:            "uid",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "UID of the file in the container. Default is 0 (root).",
						TextEdit:         textEdit("uid: ", 6, 8, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "secrets array items with a prefix",
			content: `
services:
  test:
    image: alpine
    secrets:
      - t
secrets:
  test2:
    file: ./httpd.conf`,
			line:      5,
			character: 9,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "gid",
						Detail:           types.CreateStringPointer("string"),
						TextEdit:         textEdit("gid: ", 5, 9, 1),
						Documentation:    "GID of the file in the container. Default is 0 (root).",
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "mode",
						Detail:           types.CreateStringPointer("number or string"),
						Documentation:    "File permission mode inside the container, in octal. Default is 0444 for configs and 0400 for secrets.",
						TextEdit:         textEdit("mode: ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "source",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Name of the config or secret as defined in the top-level configs or secrets section.",
						TextEdit:         textEdit("source: ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "target",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path in the container where the config or secret will be mounted. Defaults to /<source> for configs and /run/secrets/<source> for secrets.",
						TextEdit:         textEdit("target: ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:    "test2",
						TextEdit: textEdit("test2", 5, 9, 1),
					},
					{
						Label:            "uid",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "UID of the file in the container. Default is 0 (root).",
						TextEdit:         textEdit("uid: ", 5, 9, 1),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
		{
			name: "model names suggested",
			content: `
services:
  app:
    image: app
    models:
      - 
models:
  ai_model:
    model: ai/model`,
			line:      5,
			character: 8,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:    "ai_model",
						TextEdit: textEdit("ai_model", 5, 8, 0),
					},
				},
			},
		},
	}

	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "compose.yaml")), "/"))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			hub := hub.NewService()
			doc := document.NewComposeDocument(manager, uri.URI(composeFileURI), 1, []byte(tc.content))
			list, err := Completion(context.Background(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: composeFileURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, nil, &hub, doc)
			require.NoError(t, err)
			require.Equal(t, tc.list, list)
		})
	}
}

func TestCompletion_BuildStageLookups(t *testing.T) {
	dockerfileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "Dockerfile")), "/"))

	testCases := []struct {
		name              string
		dockerfileURI     string
		dockerfileContent string
		content           string
		line              uint32
		character         uint32
		list              func() *protocol.CompletionList
	}{
		{
			name:              "target attribute finds nothing",
			dockerfileContent: "FROM scratch",
			content: `
services:
  postgres:
    build:
      target: `,
			line:      4,
			character: 14,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{Items: []protocol.CompletionItem{}}
			},
		},
		{
			name:              "target attribute finds nothing",
			dockerfileContent: "FROM scratch AS",
			content: `
services:
  postgres:
    build:
      target: `,
			line:      4,
			character: 14,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{Items: []protocol.CompletionItem{}}
			},
		},
		{
			name:              "target attribute ignores target with an invalid AS",
			dockerfileContent: "FROM scratch ABC base",
			content: `
services:
  postgres:
    build:
      target: `,
			line:      4,
			character: 14,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{Items: []protocol.CompletionItem{}}
			},
		},
		{
			name:              "target attribute finds a target with uppercase AS",
			dockerfileContent: "FROM scratch AS base",
			content: `
services:
  postgres:
    build:
      target: `,
			line:      4,
			character: 14,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{
					Items: []protocol.CompletionItem{
						{
							Label:         "base",
							Documentation: "scratch",
							TextEdit:      textEdit("base", 4, 14, 0),
						},
					},
				}
			},
		},
		{
			name:              "target attribute finds a target with lowercase AS",
			dockerfileContent: "FROM scratch as base",
			content: `
services:
  postgres:
    build:
      target: `,
			line:      4,
			character: 14,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{
					Items: []protocol.CompletionItem{
						{
							Label:         "base",
							Documentation: "scratch",
							TextEdit:      textEdit("base", 4, 14, 0),
						},
					},
				}
			},
		},
		{
			name:              "target attribute finds two build stages",
			dockerfileContent: "FROM busybox as base\nFROM alpine as base2",
			content: `
services:
  postgres:
    build:
      target: `,
			line:      4,
			character: 14,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{
					Items: []protocol.CompletionItem{
						{
							Label:         "base",
							Documentation: "busybox",
							TextEdit:      textEdit("base", 4, 14, 0),
						},
						{
							Label:         "base2",
							Documentation: "alpine",
							TextEdit:      textEdit("base2", 4, 14, 0),
						},
					},
				}
			},
		},
		{
			name:              "build stage suggested by prefix",
			dockerfileContent: "FROM busybox as bstage\nFROM alpine as astage",
			content: `
services:
  postgres:
    build:
      target: a`,
			line:      4,
			character: 15,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{
					Items: []protocol.CompletionItem{
						{
							Label:         "astage",
							Documentation: "alpine",
							TextEdit:      textEdit("astage", 4, 15, 1),
						},
					},
				}
			},
		},
		{
			name:              "invalid prefix with a space inside the string is ignored",
			dockerfileContent: "FROM busybox as bstage\nFROM alpine as astage",
			content: `
services:
  postgres:
    build:
      target: a a`,
			line:      4,
			character: 17,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{Items: []protocol.CompletionItem{}}
			},
		},
		{
			name:              "invalid prefix with a space at the end of the string is ignored",
			dockerfileContent: "FROM busybox as bstage\nFROM alpine as astage",
			content: `
services:
  postgres:
    build:
      target: a `,
			line:      4,
			character: 16,
			list:      func() *protocol.CompletionList { return nil },
		},
		{
			name:              "completion in the middle with a valid prefix",
			dockerfileContent: "FROM busybox as bstage\nFROM alpine as astage",
			content: `
services:
  postgres:
    build:
      target: ab`,
			line:      4,
			character: 15,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{
					Items: []protocol.CompletionItem{
						{
							Label:         "astage",
							Documentation: "alpine",
							TextEdit:      textEdit("astage", 4, 15, 1),
						},
					},
				}
			},
		},
		{
			name:              "completion on line different from the target attribute",
			dockerfileContent: "FROM alpine as astage",
			content: `
services:
  postgres:
    build:
      target:
        `,
			line:      5,
			character: 8,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{
					Items: []protocol.CompletionItem{
						{
							Label:         "astage",
							Documentation: "alpine",
							TextEdit:      textEdit("astage", 5, 8, 0),
						},
					},
				}
			},
		},
		{
			name:              "completion on a different line from target that already has content",
			dockerfileContent: "FROM scratch AS stage",
			content: `
services:
  postgres:
    build:
      target: ab
        `,
			line:      5,
			character: 8,
			list: func() *protocol.CompletionList {
				return nil
			},
		},
		{
			name:              "no build stages suggested if dockerfile_inline used",
			dockerfileContent: "FROM scratch AS base",
			content: `
services:
  postgres:
    build:
      target: 
      dockerfile_inline: FROM scratch`,
			line:      4,
			character: 14,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{
					Items: nil,
				}
			},
		},
		{
			name:              "no build stages suggested if the dockerfile attribute is defined and invalid",
			dockerfileContent: "FROM scratch AS base",
			content: `
services:
  postgres:
    build:
      dockerfile: non-existent.txt
      target: `,
			line:      5,
			character: 14,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{Items: []protocol.CompletionItem{}}
			},
		},
		{
			name:              "build stages suggested if the dockerfile attribute is defined and valid",
			dockerfileURI:     fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "Dockerfile2")), "/")),
			dockerfileContent: "FROM scratch AS base",
			content: `
services:
  postgres:
    build:
      dockerfile: Dockerfile2
      target: `,
			line:      5,
			character: 14,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{
					Items: []protocol.CompletionItem{
						{
							Label:         "base",
							Documentation: "scratch",
							TextEdit:      textEdit("base", 5, 14, 0),
						},
					},
				}
			},
		},
		{
			name:              "build completion items include autofilled stages when build is empty",
			dockerfileContent: "FROM busybox as bstage\nFROM alpine as astage",
			content: `
services:
  postgres:
    build:
      `,
			line:      4,
			character: 6,
			list: func() *protocol.CompletionList {
				items := serviceBuildProperties(4, 6, 0)
				for i := range items {
					if items[i].Label == "target" {
						items[i].TextEdit = textEdit("target: ${1|bstage,astage|}", 4, 6, 0)
						break
					}
				}
				return &protocol.CompletionList{
					Items: items,
				}
			},
		},
		{
			name:              "build completion items include autofilled stages when build is empty when build object has other attributes",
			dockerfileContent: "FROM busybox as bstage\nFROM alpine as astage",
			content: `
services:
  postgres:
    build:
      dockerfile: Dockerfile
      `,
			line:      5,
			character: 6,
			list: func() *protocol.CompletionList {
				items := serviceBuildProperties(5, 6, 0)
				for i := range items {
					if items[i].Label == "target" {
						items[i].TextEdit = textEdit("target: ${1|bstage,astage|}", 5, 6, 0)
						break
					}
				}
				return &protocol.CompletionList{
					Items: items,
				}
			},
		},
		{
			name:              "target attribute finds nothing",
			dockerfileContent: "FROM scratch",
			content: `
services:
  postgres:
    build:
      target:  base`,
			line:      4,
			character: 14,
			list: func() *protocol.CompletionList {
				return nil
			},
		},
	}

	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "compose.yaml")), "/"))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			hub := hub.NewService()
			if tc.dockerfileContent != "" {
				u := dockerfileURI
				if tc.dockerfileURI != "" {
					u = tc.dockerfileURI
				}
				changed, err := manager.Write(context.Background(), uri.URI(u), protocol.DockerfileLanguage, 1, []byte(tc.dockerfileContent))
				require.NoError(t, err)
				require.True(t, changed)
			}
			doc := document.NewComposeDocument(manager, uri.URI(composeFileURI), 1, []byte(tc.content))
			list, err := Completion(context.Background(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: composeFileURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, manager, &hub, doc)
			require.NoError(t, err)
			require.Equal(t, tc.list(), list)
		})
	}
}

func TestCompletion_BuildStageLookups_WSL(t *testing.T) {
	testCases := []struct {
		name              string
		dockerfileContent string
		content           string
		line              uint32
		character         uint32
		list              func() *protocol.CompletionList
	}{
		{
			name:              "target attribute is null",
			dockerfileContent: "FROM scratch AS base",
			content: `
services:
  postgres:
    build:
      target: `,
			line:      4,
			character: 14,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{
					Items: []protocol.CompletionItem{
						{
							Label:         "base",
							Documentation: "scratch",
							TextEdit:      textEdit("base", 4, 14, 0),
						},
					},
				}
			},
		},
		{
			name:              "target attribute has a valid prefix",
			dockerfileContent: "FROM scratch AS base",
			content: `
services:
  postgres:
    build:
      target: b`,
			line:      4,
			character: 15,
			list: func() *protocol.CompletionList {
				return &protocol.CompletionList{
					Items: []protocol.CompletionItem{
						{
							Label:         "base",
							Documentation: "scratch",
							TextEdit:      textEdit("base", 4, 15, 1),
						},
					},
				}
			},
		},
		{
			name:              "build completion items include autofilled stages when build is empty",
			dockerfileContent: "FROM busybox as bstage\nFROM alpine as astage",
			content: `
services:
  postgres:
    build:
      `,
			line:      4,
			character: 6,
			list: func() *protocol.CompletionList {
				items := serviceBuildProperties(4, 6, 0)
				for i := range items {
					if items[i].Label == "target" {
						items[i].TextEdit = textEdit("target: ${1|bstage,astage|}", 4, 6, 0)
						break
					}
				}
				return &protocol.CompletionList{Items: items}
			},
		},
		{
			name:              "build completion items include autofilled stages when build is empty when build object has other attributes",
			dockerfileContent: "FROM busybox as bstage\nFROM alpine as astage",
			content: `
services:
  postgres:
    build:
      dockerfile: Dockerfile
      `,
			line:      5,
			character: 6,
			list: func() *protocol.CompletionList {
				items := serviceBuildProperties(5, 6, 0)
				for i := range items {
					if items[i].Label == "target" {
						items[i].TextEdit = textEdit("target: ${1|bstage,astage|}", 5, 6, 0)
						break
					}
				}
				return &protocol.CompletionList{
					Items: items,
				}
			},
		},
	}

	dockerfileURI := "file://wsl%24/docker-desktop/tmp/Dockerfile"
	composeFileURI := "file://wsl%24/docker-desktop/tmp/compose.yaml"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			hub := hub.NewService()
			changed, err := manager.Write(context.Background(), uri.URI(dockerfileURI), protocol.DockerfileLanguage, 1, []byte(tc.dockerfileContent))
			require.NoError(t, err)
			require.True(t, changed)
			doc := document.NewComposeDocument(manager, uri.URI(composeFileURI), 1, []byte(tc.content))
			list, err := Completion(context.Background(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: composeFileURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, manager, &hub, doc)
			require.NoError(t, err)
			require.Equal(t, tc.list(), list)
		})
	}
}

func TestCompletion_CustomServiceProvider(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		line      uint32
		character uint32
		list      *protocol.CompletionList
	}{
		{
			name: "type auto-suggests model",
			content: `
services:
  custom:
    provider:
      `,
			line:      4,
			character: 6,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:            "options",
						Detail:           types.CreateStringPointer("object"),
						Documentation:    "Provider-specific options.",
						TextEdit:         textEdit("options:\n        ", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "type",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "External component used by Compose to manage setup and teardown lifecycle of the service.",
						TextEdit:         textEdit("type: ${1:model}", 4, 6, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
	}

	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "compose.yaml")), "/"))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			hub := hub.NewService()
			doc := document.NewComposeDocument(manager, uri.URI(composeFileURI), 1, []byte(tc.content))
			list, err := Completion(context.Background(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: composeFileURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, manager, &hub, doc)
			require.NoError(t, err)
			require.Equal(t, tc.list, list)
		})
	}
}

func TestCompletion_NoResultExpected(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("%v-%v", t.Name(), time.Now().UnixMilli()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(dir))
	})

	fileStructure := []struct {
		name  string
		isDir bool
	}{
		{name: "a.txt", isDir: false},
		{name: "b", isDir: true},
		{name: "folder", isDir: true},
		{name: "folder/subfile.txt", isDir: false},
	}
	for _, entry := range fileStructure {
		if entry.isDir {
			require.NoError(t, os.Mkdir(filepath.Join(dir, entry.name), 0o755))
		} else {
			f, err := os.Create(filepath.Join(dir, entry.name))
			require.NoError(t, err)
			require.NoError(t, f.Close())
		}
	}

	testCases := []struct {
		name      string
		content   string
		line      uint32
		character uint32
		list      *protocol.CompletionList
	}{
		{
			name: "./ prefix ignored on other attributes",
			content: `
services:
  test:
    depends_on:
      - ./`,
			line:      4,
			character: 10,
			list:      nil,
		},
		{
			name: "suggests nothing for source attribute with no type specified",
			content: `
services:
  test:
    volumes:
      - source: `,
			line:      4,
			character: 16,
			list:      nil,
		},
		{
			name: "suggests nothing for source attribute when type is not bind",
			content: `
services:
  test:
    volumes:
      - type: volume
        source: `,
			line:      5,
			character: 16,
			list:      nil,
		},
		{
			name: "suggests nothing for multiple volumes on a non-bind one",
			content: `
services:
  test:
    volumes:
      - type: bind
      - source: ./`,
			line:      5,
			character: 18,
			list:      nil,
		},
		{
			name: "suggests nothing when volumes not under a services node",
			content: `
services2:
  test:
    volumes:
      - ./`,
			line:      4,
			character: 10,
			list:      nil,
		},
		{
			name: "suggests nothing when volumes object not under a services node",
			content: `
services2:
  test:
    volumes:
      - type: bind
        source: `,
			line:      5,
			character: 16,
			list:      nil,
		},
		{
			name: "path attribute under an env_file object should suggest nothing as it should be an array item",
			content: `
services:
  test:
    env_file: 
      path: `,
			line:      4,
			character: 12,
			list:      nil,
		},
	}

	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(dir, "compose.yaml")), "/"))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			hub := hub.NewService()
			doc := document.NewComposeDocument(manager, uri.URI(composeFileURI), 1, []byte(tc.content))
			list, err := Completion(context.Background(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: composeFileURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, manager, &hub, doc)
			require.NoError(t, err)
			require.Equal(t, tc.list, list)
		})
	}
}

func TestCompletion_VolumeFolderListing(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("%v-%v", t.Name(), time.Now().UnixMilli()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(dir))
	})

	fileStructure := []struct {
		name  string
		isDir bool
	}{
		{name: "a.txt", isDir: false},
		{name: "b", isDir: true},
		{name: "folder", isDir: true},
		{name: "folder/subfile.txt", isDir: false},
	}
	for _, entry := range fileStructure {
		if entry.isDir {
			require.NoError(t, os.Mkdir(filepath.Join(dir, entry.name), 0o755))
		} else {
			f, err := os.Create(filepath.Join(dir, entry.name))
			require.NoError(t, err)
			require.NoError(t, f.Close())
		}
	}

	allItems := &protocol.CompletionList{
		Items: []protocol.CompletionItem{
			{
				Label: "a.txt",
				Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFile),
			},
			{
				Label: "b",
				Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
			},
			{
				Label: "folder",
				Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
			},
		},
	}

	testCases := []struct {
		name      string
		content   string
		line      uint32
		character uint32
		list      *protocol.CompletionList
	}{
		{
			name: "./ prefix ignored on other attributes",
			content: `
services:
  test:
    depends_on:
      - ./`,
			line:      4,
			character: 10,
			list:      nil,
		},
		{
			name: "suggests nothing for source attribute with no type specified",
			content: `
services:
  test:
    volumes:
      - source: `,
			line:      4,
			character: 16,
			list:      nil,
		},
		{
			name: "suggests nothing for source attribute when type is not bind",
			content: `
services:
  test:
    volumes:
      - type: volume
        source: `,
			line:      5,
			character: 16,
			list:      nil,
		},
		{
			name: "suggests nothing for multiple volumes on a non-bind one",
			content: `
services:
  test:
    volumes:
      - type: bind
      - source: ./`,
			line:      5,
			character: 18,
			list:      nil,
		},
		{
			name: "suggests nothing when volumes not under a services node",
			content: `
services2:
  test:
    volumes:
      - ./`,
			line:      4,
			character: 10,
			list:      nil,
		},
		{
			name: "suggests nothing when volumes object not under a services node",
			content: `
services2:
  test:
    volumes:
      - type: bind
        source: `,
			line:      5,
			character: 16,
			list:      nil,
		},
		{
			name: "path attribute under an env_file object should suggest nothing as it should be an array item",
			content: `
services:
  test:
    env_file: 
      path: `,
			line:      4,
			character: 12,
			list:      nil,
		},
		{
			name: "suggest file structure with a ./ prefix",
			content: `
services:
  test:
    volumes:
      - ./`,
			line:      4,
			character: 10,
			list:      allItems,
		},
		{
			name: "suggest file structure with a quoted \"./\" prefix",
			content: `
services:
  test:
    volumes:
      - "./"`,
			line:      4,
			character: 11,
			list:      allItems,
		},
		{
			name: "suggest file structure with a quoted './' prefix",
			content: `
services:
  test:
    volumes:
      - './'`,
			line:      4,
			character: 11,
			list:      allItems,
		},
		{
			name: "suggest file structure with a ./a prefix",
			content: `
services:
  test:
    volumes:
      - ./a`,
			line:      4,
			character: 11,
			list:      allItems,
		},
		{
			name: "suggest file structure with a ./folder/ prefix",
			content: `
services:
  test:
    volumes:
      - ./folder/`,
			line:      4,
			character: 17,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "subfile.txt",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFile),
					},
				},
			},
		},
		{
			name: "suggests file structure for source attribute when type is bind with no content",
			content: `
services:
  test:
    volumes:
      - type: bind
        source: `,
			line:      5,
			character: 16,
			list:      allItems,
		},
		{
			name: "suggests file structure for source attribute when type is bind with a ./ prefix",
			content: `
services:
  test:
    volumes:
      - type: bind
        source: ./`,
			line:      5,
			character: 18,
			list:      allItems,
		},
		{
			name: "suggests inner file structure for source attribute when type is bind with a ./folder/ prefix",
			content: `
services:
  test:
    volumes:
      - type: bind
        source: ./folder/`,
			line:      5,
			character: 25,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "subfile.txt",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFile),
					},
				},
			},
		},
	}

	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(dir, "compose.yaml")), "/"))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			hub := hub.NewService()
			doc := document.NewComposeDocument(manager, uri.URI(composeFileURI), 1, []byte(tc.content))
			list, err := Completion(context.Background(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: composeFileURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, manager, &hub, doc)
			require.NoError(t, err)
			require.Equal(t, tc.list, list)
		})
	}
}

func testCompletion_RootFolder(t *testing.T, composeFileURI, rootDir, content string, line, character protocol.UInteger) {
	entries, err := os.ReadDir(rootDir)
	require.NoError(t, err, "could not read root directory %v", rootDir)
	require.True(t, len(entries) > 0, "no entries in %v which makes it hard to validate this test", rootDir)
	items := make([]protocol.CompletionItem, len(entries))
	for i := range entries {
		items[i] = protocol.CompletionItem{Label: entries[i].Name()}
		if entries[i].IsDir() {
			items[i].Kind = types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder)
		} else if entries[i].Type() == os.ModeSymlink {
			items[i].Kind = types.CreateCompletionItemKindPointer(protocol.CompletionItemKindReference)
		} else {
			items[i].Kind = types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFile)
		}
	}

	manager := document.NewDocumentManager()
	hub := hub.NewService()
	doc := document.NewComposeDocument(manager, uri.URI(composeFileURI), 1, []byte(content))
	list, err := Completion(context.Background(), &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: composeFileURI},
			Position:     protocol.Position{Line: line, Character: character},
		},
	}, manager, &hub, doc)
	require.NoError(t, err)
	require.Equal(t, &protocol.CompletionList{Items: items}, list)
}

func TestCompletion_RootFolder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows as / is not a root folder on Windows")
		return
	}

	composeFileURI := fmt.Sprintf("file://%v", filepath.Join(os.TempDir(), "compose.yaml"))
	testCompletion_RootFolder(t, composeFileURI, "/", `
services:
  s:
    volumes:
      - /`, 4, 9)
}

func TestCompletion_WindowsRootFolder(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping on non-Windows systems as it will not have a C:\\ folder")
		return
	}

	composeFileURI := fmt.Sprintf("file:///%v", filepath.Join(os.TempDir(), "compose.yaml"))
	testCompletion_RootFolder(t, composeFileURI, "C:\\", `
services:
  s:
    volumes:
      - C:\`, 4, 11)
}

func TestCompletion_FileStructure(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		hideFiles bool
		line      uint32
		character uint32
	}{
		{
			name: "services - context attribute under a build object",
			content: `
services:
  test:
    build: 
      context: `,
			hideFiles: true,
			line:      4,
			character: 15,
		},
		{
			name: "services - dockerfile attribute under a build object",
			content: `
services:
  test:
    build: 
      dockerfile: `,
			hideFiles: false,
			line:      4,
			character: 18,
		},
		{
			name: "services - file attribute under a credential_spec object",
			content: `
services:
  test:
    credential_spec: 
      file: `,
			hideFiles: false,
			line:      4,
			character: 12,
		},
		{
			name: "services - env_file string attribute",
			content: `
services:
  test:
    env_file: `,
			hideFiles: false,
			line:      3,
			character: 14,
		},
		{
			name: "services - env_file array item as a string attribute",
			content: `
services:
  test:
    env_file: 
      - `,
			hideFiles: false,
			line:      4,
			character: 8,
		},
		{
			name: "services - path attribute of an env_file array object",
			content: `
services:
  test:
    env_file: 
      - path: `,
			hideFiles: false,
			line:      4,
			character: 14,
		},
		{
			name: "services - file attribute of an extends object",
			content: `
services:
  test:
    extends: 
      file: `,
			hideFiles: false,
			line:      4,
			character: 12,
		},
		{
			name: "services - label_file string attribute",
			content: `
services:
  test:
    label_file: `,
			hideFiles: false,
			line:      3,
			character: 16,
		},
		{
			name: "services - label_file array item as a string attribute",
			content: `
services:
  test:
    label_file: 
      - `,
			hideFiles: false,
			line:      4,
			character: 8,
		},
		{
			name: "configs - file attribute",
			content: `
configs:
  test:
    file: `,
			hideFiles: false,
			line:      3,
			character: 10,
		},
		{
			name: "secrets - file attribute",
			content: `
secrets:
  test:
    file: `,
			hideFiles: false,
			line:      3,
			character: 10,
		},
		{
			name: "include - env_file attribute",
			content: `
include:
  - env_file: `,
			hideFiles: false,
			line:      2,
			character: 14,
		},
		{
			name: "include - env_file attribute's string array items",
			content: `
include:
  - env_file:
    - `,
			hideFiles: false,
			line:      3,
			character: 6,
		},
		{
			name: "include - path attribute",
			content: `
include:
  - path: `,
			hideFiles: false,
			line:      2,
			character: 10,
		},
		{
			name: "include - path attribute's string array items",
			content: `
include:
  - path:
    - `,
			hideFiles: false,
			line:      3,
			character: 6,
		},
	}

	setups := []struct {
		description  string
		content      string
		offset       uint32
		result       *protocol.CompletionList
		folderResult *protocol.CompletionList
	}{
		{
			description: "no prefix",
			content:     "",
			offset:      0,
			result: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "a.txt",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFile),
					},
					{
						Label: "b",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
					{
						Label: "folder",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
				},
			},
			folderResult: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "b",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
					{
						Label: "folder",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
				},
			},
		},
		{
			description: "./ prefix",
			content:     "./",
			offset:      2,
			result: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "a.txt",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFile),
					},
					{
						Label: "b",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
					{
						Label: "folder",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
				},
			},
			folderResult: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "b",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
					{
						Label: "folder",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
				},
			},
		},
		{
			description: "./folder/ prefix",
			content:     "./folder/",
			offset:      9,
			result: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "subfile.txt",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFile),
					},
				},
			},
			folderResult: nil,
		},
	}

	dir := createFileStructure(t)
	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(dir, "compose.yaml")), "/"))

	for _, tc := range testCases {
		for _, setup := range setups {
			t.Run(fmt.Sprintf("%v (%v)", tc.name, setup.description), func(t *testing.T) {
				manager := document.NewDocumentManager()
				hub := hub.NewService()
				doc := document.NewComposeDocument(manager, uri.URI(composeFileURI), 1, []byte(tc.content+setup.content))
				list, err := Completion(context.Background(), &protocol.CompletionParams{
					TextDocumentPositionParams: protocol.TextDocumentPositionParams{
						TextDocument: protocol.TextDocumentIdentifier{URI: composeFileURI},
						Position:     protocol.Position{Line: tc.line, Character: tc.character + setup.offset},
					},
				}, manager, &hub, doc)
				require.NoError(t, err)
				if tc.hideFiles {
					require.Equal(t, setup.folderResult, list)
				} else {
					require.Equal(t, setup.result, list)
				}
			})
		}
	}
}

func TestCompletion_FileStructureMerged(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		line      uint32
		character uint32
		list      *protocol.CompletionList
	}{
		{
			name: "include array item suggests attributes and file structure",
			content: `
include:
  - `,
			line:      2,
			character: 4,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "a.txt",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFile),
					},
					{
						Label: "b",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
					{
						Label:            "env_file",
						Detail:           types.CreateStringPointer("array or string"),
						Documentation:    "Either a single string or a list of strings.",
						TextEdit:         textEdit("env_file:", 2, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label: "folder",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
					{
						Label:            "path",
						Detail:           types.CreateStringPointer("array or string"),
						Documentation:    "Either a single string or a list of strings.",
						TextEdit:         textEdit("path:", 2, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
					{
						Label:            "project_directory",
						Detail:           types.CreateStringPointer("string"),
						Documentation:    "Path to resolve relative paths set in the Compose file",
						TextEdit:         textEdit("project_directory: ", 2, 4, 0),
						InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
						InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					},
				},
			},
		},
	}

	dir := createFileStructure(t)
	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(dir, "compose.yaml")), "/"))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			hub := hub.NewService()
			doc := document.NewComposeDocument(manager, uri.URI(composeFileURI), 1, []byte(tc.content))
			list, err := Completion(context.Background(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: composeFileURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, manager, &hub, doc)
			require.NoError(t, err)
			require.Equal(t, tc.list, list)
		})
	}
}

func TestCompletion_ImageTags(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		line      uint32
		character uint32
		list      *protocol.CompletionList
	}{
		{
			name: "docker/lsp:g",
			content: `
services:
  test:
    image: docker/lsp:g`,
			line:      3,
			character: 23,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "golang",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindModule),
					},
				},
			},
		},
		{
			name: "'docker/lsp:g'",
			content: `
services:
  test:
    image: 'docker/lsp:g'`,
			line:      3,
			character: 24,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "golang",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindModule),
					},
				},
			},
		},
		{
			name: "\"docker/lsp:g\"",
			content: `
services:
  test:
    image: "docker/lsp:g"`,
			line:      3,
			character: 24,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "golang",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindModule),
					},
				},
			},
		},
		{
			name: "ubuntu:20.",
			content: `
services:
  test:
    image: ubuntu:20.`,
			line:      3,
			character: 21,
			list: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "20.04",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindModule),
					},
					{
						Label: "20.10",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindModule),
					},
				},
			},
		},
		{
			name: "text is correct but cursor not in the right place",
			content: `
services:
  test:
    image: docker/lsp:g`,
			line:      3,
			character: 15,
			list:      nil,
		},
	}

	dir := createFileStructure(t)
	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(dir, "compose.yaml")), "/"))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			hub := hub.NewService()
			doc := document.NewComposeDocument(manager, uri.URI(composeFileURI), 1, []byte(tc.content))
			list, err := Completion(context.Background(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: composeFileURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, manager, &hub, doc)
			require.NoError(t, err)
			require.Equal(t, tc.list, list)
		})
	}
}

func textEdit(newText string, line, character, prefixLength protocol.UInteger) protocol.TextEdit {
	return protocol.TextEdit{
		NewText: newText,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      line,
				Character: character - prefixLength,
			},
			End: protocol.Position{
				Line:      line,
				Character: character,
			},
		},
	}
}

func createFileStructure(t *testing.T) string {
	dir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("%v-%v", t.Name(), time.Now().UnixMilli()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(dir))
	})

	fileStructure := []struct {
		name  string
		isDir bool
	}{
		{name: "a.txt", isDir: false},
		{name: "b", isDir: true},
		{name: "folder", isDir: true},
		{name: "folder/subfile.txt", isDir: false},
	}
	for _, entry := range fileStructure {
		if entry.isDir {
			require.NoError(t, os.Mkdir(filepath.Join(dir, entry.name), 0o755))
		} else {
			f, err := os.Create(filepath.Join(dir, entry.name))
			require.NoError(t, err)
			require.NoError(t, f.Close())
		}
	}
	return dir
}
