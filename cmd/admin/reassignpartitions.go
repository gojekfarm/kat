package admin

import (
	"github.com/gojek/kat/cmd/base"
	"github.com/gojek/kat/logger"
	"github.com/gojek/kat/pkg/client"
	"github.com/spf13/cobra"
)

type reassignPartitions struct {
	client.Lister
	client.Partitioner
	topics             string
	brokerIds          string
	batch              int
	timeoutPerBatchInS int
	pollIntervalInS    int
	throttle           int
}

var ReassignPartitionsCmd = &cobra.Command{
	Use:   "reassign-partitions",
	Short: "Reassigns the partitions for topics",
	Run: func(command *cobra.Command, args []string) {
		cobraUtil := base.NewCobraUtil(command)
		zookeeper := cobraUtil.GetStringArg("zookeeper")
		baseCmd := base.Init(cobraUtil, base.WithPartition(zookeeper))
		r := reassignPartitions{Lister: baseCmd.GetTopic(), Partitioner: baseCmd.GetPartition(), topics: cobraUtil.GetStringArg("topics"),
			brokerIds: cobraUtil.GetStringArg("broker-ids"), batch: cobraUtil.GetIntArg("batch"),
			timeoutPerBatchInS: cobraUtil.GetIntArg("timeout-per-batch"), pollIntervalInS: cobraUtil.GetIntArg("status-poll-interval"),
			throttle: cobraUtil.GetIntArg("throttle")}
		r.reassignPartitions()
	},
}

func init() {
	ReassignPartitionsCmd.PersistentFlags().StringP("topics", "t", "",
		"Regex to match the topics that require partition reassignment. eg: \".*\", \"test-.*-topic\", \"topic1|topic2\"")
	ReassignPartitionsCmd.PersistentFlags().StringP("zookeeper", "z", "", "Comma separated list of zookeeper ips")
	ReassignPartitionsCmd.PersistentFlags().StringP("broker-ids", "i", "", "Comma separated list of broker ids. eg: \"1,2,3,4,5,6\"")
	ReassignPartitionsCmd.PersistentFlags().IntP("batch", "", 1, "Batch size to split reassignment")
	ReassignPartitionsCmd.PersistentFlags().IntP("timeout-per-batch", "", 300, "Timeout for reassignment per batch in seconds")
	ReassignPartitionsCmd.PersistentFlags().IntP("status-poll-interval", "", 5, "Interval in seconds for polling for reassignment status")
	ReassignPartitionsCmd.PersistentFlags().IntP("throttle", "", 10000000, "Throttle for reassignment in bytes/sec")
	if err := ReassignPartitionsCmd.MarkPersistentFlagRequired("topics"); err != nil {
		logger.Fatal(err)
	}
	if err := ReassignPartitionsCmd.MarkPersistentFlagRequired("zookeeper"); err != nil {
		logger.Fatal(err)
	}
	if err := ReassignPartitionsCmd.MarkPersistentFlagRequired("broker-ids"); err != nil {
		logger.Fatal(err)
	}
}

func (r *reassignPartitions) reassignPartitions() {
	topics, err := r.ListOnly(r.topics, true)
	if err != nil {
		logger.Fatalf("Error while filtering topics - %v\n", err)
	}

	if len(topics) == 0 {
		logger.Infof("Did not find any topic matching - %v\n", r.topics)
		return
	}

	err = r.ReassignPartitions(topics, r.brokerIds, r.batch, r.timeoutPerBatchInS, r.pollIntervalInS, r.throttle)
	if err != nil {
		logger.Errorf("Error while reassigning partitions: %s", err)
		return
	}
	logger.Info("Successfully reassigned partitions")
}
