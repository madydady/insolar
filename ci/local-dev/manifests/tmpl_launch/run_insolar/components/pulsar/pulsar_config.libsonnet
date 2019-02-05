local import_params = import '../params.libsonnet';
local params = import_params.global.utils;

{
	"pulsar": {
		"connectiontype": "tcp",
		"mainlisteneraddress": "0.0.0.0:58090",
		"storage": {
			"datadirectory": "/opt/insolar/pulsar",
			"txretriesonconflict": 0
		},
		"pulsetime": 10000,
		"receivingsigntimeout": 1000,
		"receivingnumbertimeout": 1000,
		"receivingvectortimeout": 1000,
		"receivingsignsforchosentimeout": 0,
		"neighbours": [],
		"numberofrandomhosts": 1,
		"numberdelta": 10,
		"distributiontransport": {
			"protocol": "TCP",
			"address": "0.0.0.0:58091",
			"behindnat": false
		},
		"pulsedistributor": {
			"bootstraphosts": [
				params.host_template % id for id in std.range(0, params.get_num_nodes - 1)
			]
		}
	},
	"keyspath": "/opt/insolar/config/bootstrap_keys.json",
	"log": {
		"level": "Debug",
		"adapter": "logrus"
	}
}