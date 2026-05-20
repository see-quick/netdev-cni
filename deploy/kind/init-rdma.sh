#!/bin/sh
# Run inside kind worker node to load rdma_rxe for RoCE simulation.
set -e
modprobe rdma_rxe || echo "rdma_rxe not available (kernel may not support it)"
IF=$(ip link show | grep -E 'eth[0-9]' | head -1 | awk '{print $2}' | tr -d ':')
if [ -n "$IF" ]; then
  rdma link add rxe0 type rxe netdev "$IF" 2>/dev/null || true
  echo "rdma_rxe: created rxe0 over $IF"
fi
