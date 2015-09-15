/*
Copyright (c) 2014 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pool

import (
	"flag"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/vmware/govmomi/govc/cli"
	"github.com/vmware/govmomi/govc/flags"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

type info struct {
	*flags.DatacenterFlag
	*flags.OutputFlag
}

func init() {
	cli.Register("pool.info", &info{})
}

func (cmd *info) Register(f *flag.FlagSet) {}

func (cmd *info) Process() error { return nil }

func (cmd *info) Usage() string {
	return "POOL..."
}

func (cmd *info) Description() string {
	return "Retrieve information about one or more resource POOLs.\n" + poolNameHelp
}

func (cmd *info) Run(f *flag.FlagSet) error {
	if f.NArg() == 0 {
		return flag.ErrHelp
	}

	c, err := cmd.Client()
	if err != nil {
		return err
	}

	ctx := context.TODO()

	finder, err := cmd.Finder()
	if err != nil {
		return err
	}

	var res infoResult
	var props []string

	if cmd.OutputFlag.JSON {
		props = nil
	} else {
		props = []string{
			"name",
			"config.cpuAllocation",
			"config.memoryAllocation",
			"runtime.cpu",
			"runtime.memory",
		}
	}

	for _, arg := range f.Args() {
		objects, err := finder.ResourcePoolList(ctx, arg)
		if err != nil {
			return err
		}
		res.objects = append(res.objects, objects...)
	}

	if len(res.objects) != 0 {
		refs := make([]types.ManagedObjectReference, 0, len(res.objects))
		for _, o := range res.objects {
			refs = append(refs, o.Reference())
		}

		pc := property.DefaultCollector(c)
		err = pc.Retrieve(ctx, refs, props, &res.ResourcePools)
		if err != nil {
			return err
		}
	}

	return cmd.WriteResult(&res)
}

type infoResult struct {
	ResourcePools []mo.ResourcePool
	objects       []*object.ResourcePool
}

func (r *infoResult) Write(w io.Writer) error {
	tw := tabwriter.NewWriter(w, 2, 0, 2, ' ', 0)

	for _, pool := range r.ResourcePools {
		fmt.Fprintf(tw, "Name:\t%s\n", pool.Name)

		writeInfo(tw, "CPU", "MHz", &pool.Runtime.Cpu, pool.Config.CpuAllocation)
		writeInfo(tw, "Mem", "MB", &pool.Runtime.Memory, pool.Config.MemoryAllocation)
	}

	return tw.Flush()
}

func writeInfo(w io.Writer, name string, units string, ru *types.ResourcePoolResourceUsage, b types.BaseResourceAllocationInfo) {
	ra := b.GetResourceAllocationInfo()
	usage := 100.0 * float64(ru.OverallUsage) / float64(ru.MaxUsage)
	shares := ""
	limit := "unlimited"

	if ra.Shares.Level == types.SharesLevelCustom {
		shares = fmt.Sprintf(" (%d)", ra.Shares.Shares)
	}

	if ra.Limit != -1 {
		limit = fmt.Sprintf("%d%s", ra.Limit, units)
	}

	fmt.Fprintf(w, "  %s Usage:\t%d%s (%0.1f%%)\n", name, ru.OverallUsage, units, usage)
	fmt.Fprintf(w, "  %s Shares:\t%s%s\n", name, ra.Shares.Level, shares)
	fmt.Fprintf(w, "  %s Reservation:\t%d%s (expandable=%v)\n", name, ra.Reservation, units, *ra.ExpandableReservation)
	fmt.Fprintf(w, "  %s Limit:\t%s\n", name, limit)
}
