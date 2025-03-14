// Copyright (C) 2014-2018 Wutong Co., Ltd.
// WUTONG, Application Management Platform

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong,
// one or multiple Commercial Licenses authorized by Wutong Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package etcd

import (
	"fmt"

	"go.etcd.io/etcd/api/v3/mvccpb"
	v3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
)

// Queue implements a multi-reader, multi-writer distributed queue.
type Queue struct {
	client *v3.Client
	ctx    context.Context

	keyPrefix string
}

// NewQueue new queue
func NewQueue(ctx context.Context, client *v3.Client, keyPrefix string) *Queue {
	return &Queue{client, ctx, keyPrefix}
}

// Enqueue en queue
func (q *Queue) Enqueue(val string) error {
	_, err := newUniqueKV(q.ctx, q.client, q.keyPrefix, val)
	return err
}

// Dequeue returns Enqueue()'d elements in FIFO order. If the
// queue is empty, Dequeue blocks until elements are available.
func (q *Queue) Dequeue() (string, error) {
	for {
		// TODO: fewer round trips by fetching more than one key
		resp, err := q.client.Get(q.ctx, q.keyPrefix, v3.WithFirstRev()...)
		if err != nil {
			return "", err
		}

		kv, err := claimFirstKey(q.ctx, q.client, resp.Kvs)
		if err != nil {
			return "", err
		} else if kv != nil {
			return string(kv.Value), nil
		} else if resp.More {
			// missed some items, retry to read in more
			return q.Dequeue()
		}

		// nothing yet; wait on elements
		ev, err := WaitPrefixEvents(
			q.client,
			q.keyPrefix,
			resp.Header.Revision,
			[]mvccpb.Event_EventType{mvccpb.PUT})
		if err != nil {
			if err == ErrNoUpdateForLongTime {
				continue
			}
			return "", err
		}
		if ev == nil {
			return "", fmt.Errorf("event is nil")
		}
		if ev.Kv == nil {
			return "", fmt.Errorf("event key value is nil")
		}
		ok, err := deleteRevKey(q.ctx, q.client, string(ev.Kv.Key), ev.Kv.ModRevision)
		if err != nil {
			return "", err
		} else if !ok {
			return q.Dequeue()
		}
		return string(ev.Kv.Value), err
	}
}
