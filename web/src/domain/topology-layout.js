function topologyKind(node) {
  return String(node?.kind || "").toLowerCase();
}
function topologyLayer(node) {
  const kind = topologyKind(node);
  if (
    [
      "pseudo-internet",
      "pseudo-external-endpoint",
      "ingress",
      "gateway",
      "gatewayroute",
      "service",
    ].includes(kind)
  )
    return "application";
  if (["namespace", "workload"].includes(kind)) return "workload";
  if (["endpoint-slice", "pod", "node"].includes(kind)) return "runtime";
  return "infrastructure";
}
export function topologyCondensedHierarchy(nodes, edges) {
  const ordered = [...nodes].sort((a, b) => a.id.localeCompare(b.id)),
    ids = new Set(ordered.map((node) => node.id)),
    adjacency = new Map(ordered.map((node) => [node.id, []]));
  edges.forEach((edge) => {
    if (
      ids.has(edge.source) &&
      ids.has(edge.target) &&
      edge.source !== edge.target
    )
      adjacency.get(edge.source).push(edge.target);
  });
  adjacency.forEach((targets) => targets.sort());
  let cursor = 0;
  const indices = new Map(),
    lowlink = new Map(),
    stack = [],
    onStack = new Set(),
    rawComponents = [];
  const visit = (id) => {
    indices.set(id, cursor);
    lowlink.set(id, cursor++);
    stack.push(id);
    onStack.add(id);
    (adjacency.get(id) || []).forEach((target) => {
      if (!indices.has(target)) {
        visit(target);
        lowlink.set(id, Math.min(lowlink.get(id), lowlink.get(target)));
      } else if (onStack.has(target))
        lowlink.set(id, Math.min(lowlink.get(id), indices.get(target)));
    });
    if (lowlink.get(id) !== indices.get(id)) return;
    const members = [];
    let member = "";
    do {
      member = stack.pop();
      onStack.delete(member);
      members.push(member);
    } while (member !== id);
    rawComponents.push(members.sort());
  };
  ordered.forEach((node) => {
    if (!indices.has(node.id)) visit(node.id);
  });
  const components = rawComponents.sort((a, b) => a[0].localeCompare(b[0])),
    componentOf = new Map();
  components.forEach((members, index) =>
    members.forEach((id) => componentOf.set(id, index)),
  );
  const outgoing = new Map(components.map((_, index) => [index, new Set()])),
    incoming = new Map(components.map((_, index) => [index, new Set()]));
  edges.forEach((edge) => {
    const source = componentOf.get(edge.source),
      target = componentOf.get(edge.target);
    if (source === undefined || target === undefined || source === target)
      return;
    outgoing.get(source).add(target);
    incoming.get(target).add(source);
  });
  const depth = new Map(components.map((_, index) => [index, 0])),
    indegree = new Map(
      components.map((_, index) => [index, incoming.get(index).size]),
    ),
    queue = components
      .map((_, index) => index)
      .filter((index) => indegree.get(index) === 0)
      .sort((a, b) => components[a][0].localeCompare(components[b][0])),
    order = [];
  for (let index = 0; index < queue.length; index++) {
    const source = queue[index];
    order.push(source);
    [...outgoing.get(source)]
      .sort((a, b) => components[a][0].localeCompare(components[b][0]))
      .forEach((target) => {
        depth.set(target, Math.max(depth.get(target), depth.get(source) + 1));
        indegree.set(target, indegree.get(target) - 1);
        if (indegree.get(target) === 0) queue.push(target);
      });
  }
  const leaves = new Map();
  [...order].reverse().forEach((index) => {
    const children = [...outgoing.get(index)],
      reachable = new Set();
    if (!children.length) reachable.add(index);
    else
      children.forEach((child) =>
        (leaves.get(child) || new Set([child])).forEach((leaf) =>
          reachable.add(leaf),
        ),
      );
    leaves.set(index, reachable);
  });
  return { components, componentOf, outgoing, incoming, depth, leaves };
}

export function topologyWeakComponents(nodes, edges) {
  const nodeById = new Map(nodes.map((node) => [node.id, node])),
    adjacency = new Map(nodes.map((node) => [node.id, new Set()]));
  edges.forEach((edge) => {
    if (
      !nodeById.has(edge.source) ||
      !nodeById.has(edge.target) ||
      edge.source === edge.target
    )
      return;
    adjacency.get(edge.source).add(edge.target);
    adjacency.get(edge.target).add(edge.source);
  });
  const visited = new Set(),
    connected = [],
    isolates = [];
  [...nodes]
    .sort((a, b) => a.id.localeCompare(b.id))
    .forEach((node) => {
      if (visited.has(node.id)) return;
      if (!adjacency.get(node.id).size) {
        visited.add(node.id);
        isolates.push(node);
        return;
      }
      const queue = [node.id],
        members = [];
      visited.add(node.id);
      for (let cursor = 0; cursor < queue.length; cursor++) {
        const id = queue[cursor];
        members.push(nodeById.get(id));
        [...adjacency.get(id)].sort().forEach((next) => {
          if (!visited.has(next)) {
            visited.add(next);
            queue.push(next);
          }
        });
      }
      connected.push(members);
    });
  return { connected, isolates };
}

export function topologyLayoutIsolates(nodes, nodeW, nodeH) {
  const columns = Math.max(1, Math.ceil(Math.sqrt(nodes.length))),
    gapX = 72,
    gapY = 48,
    drawn = new Map();
  nodes.forEach((node, index) => {
    const column = index % columns,
      row = Math.floor(index / columns);
    drawn.set(node.id, {
      ...node,
      x: column * (nodeW + gapX),
      y: row * (nodeH + gapY),
      w: nodeW,
      h: nodeH,
      rank: 0,
      isolate: true,
    });
  });
  const rows = Math.ceil(nodes.length / columns);
  return {
    drawn,
    width: columns * nodeW + Math.max(0, columns - 1) * gapX,
    height: rows * nodeH + Math.max(0, rows - 1) * gapY,
  };
}

export function topologyLayoutComponent(nodes, edges, nodeW, nodeH) {
  const hierarchy = topologyCondensedHierarchy(nodes, edges),
    semanticOrder = (node) => {
      const kind = topologyKind(node);
      if (kind.startsWith("pseudo-")) return 0;
      if (["ingress", "gateway", "gatewayroute"].includes(kind)) return 1;
      if (kind === "service") return 2;
      if (kind === "workload") return 3;
      return topologyLayer(node) === "runtime" ? 4 : 5;
    },
    stable = (a, b) =>
      semanticOrder(a) - semanticOrder(b) ||
      a.name.localeCompare(b.name) ||
      a.id.localeCompare(b.id),
    rows = new Map(),
    nodeById = new Map(nodes.map((node) => [node.id, node]));
  nodes.forEach((node) => {
    const rank = hierarchy.depth.get(hierarchy.componentOf.get(node.id)) || 0;
    if (!rows.has(rank)) rows.set(rank, []);
    rows.get(rank).push(node);
  });
  rows.forEach((items) => items.sort(stable));
  const parents = new Map(nodes.map((node) => [node.id, []])),
    children = new Map(nodes.map((node) => [node.id, []]));
  edges.forEach((edge) => {
    if (
      !nodeById.has(edge.source) ||
      !nodeById.has(edge.target) ||
      edge.source === edge.target
    )
      return;
    parents.get(edge.target).push(edge.source);
    children.get(edge.source).push(edge.target);
  });
  const ranks = [...rows.keys()].sort((a, b) => a - b),
    position = () => {
      const result = new Map();
      rows.forEach((items) =>
        items.forEach((node, index) => result.set(node.id, index)),
      );
      return result;
    },
    reorder = (rank, neighbors, positions) => {
      const items = rows.get(rank) || [];
      items.sort((a, b) => {
        const score = (node) => {
            const values = neighbors
              .get(node.id)
              .map((id) => positions.get(id))
              .filter(Number.isFinite);
            return values.length
              ? values.reduce((sum, value) => sum + value, 0) / values.length
              : Number.POSITIVE_INFINITY;
          },
          delta = score(a) - score(b);
        return Number.isFinite(delta) && Math.abs(delta) > 0.001
          ? delta
          : stable(a, b);
      });
    };
  for (let pass = 0; pass < 4; pass++) {
    let positions = position();
    ranks.slice(1).forEach((rank) => {
      reorder(rank, parents, positions);
      positions = position();
    });
    ranks
      .slice(0, -1)
      .reverse()
      .forEach((rank) => {
        reorder(rank, children, positions);
        positions = position();
      });
  }
  const drawn = new Map(),
    nodeGap = 72,
    rowGap = 48,
    rankGap = 112,
    maxColumns = Math.max(1, Math.ceil(Math.sqrt(nodes.length * 1.6)));
  let bandY = 0,
    width = 0;
  ranks.forEach((rank) => {
    const items = rows.get(rank),
      columns = Math.min(items.length, maxColumns),
      localRows = Math.ceil(items.length / columns),
      rowWidth = columns * nodeW + Math.max(0, columns - 1) * nodeGap;
    width = Math.max(width, rowWidth);
    items.forEach((node, index) => {
      const column = index % columns,
        localRow = Math.floor(index / columns);
      drawn.set(node.id, {
        ...node,
        x: column * (nodeW + nodeGap),
        y: bandY + localRow * (nodeH + rowGap),
        w: nodeW,
        h: nodeH,
        rank,
      });
    });
    bandY += localRows * nodeH + Math.max(0, localRows - 1) * rowGap + rankGap;
  });
  const height = Math.max(nodeH, bandY - rankGap);
  return { drawn, width: Math.max(nodeW, width), height };
}

export function topologyOverviewLayout(nodes, edges, width, nodeW, nodeH) {
  const { connected, isolates } = topologyWeakComponents(nodes, edges),
    nodeIDs = (members) => new Set(members.map((node) => node.id)),
    layouts = connected
      .map((members) => {
        const ids = nodeIDs(members),
          componentEdges = edges.filter(
            (edge) => ids.has(edge.source) && ids.has(edge.target),
          ),
          layout = topologyLayoutComponent(
            members,
            componentEdges,
            nodeW,
            nodeH,
          );
        return { ...layout, key: members[0]?.id || "" };
      })
      .sort((a, b) => a.key.localeCompare(b.key)),
    margin = 52,
    componentGap = 128,
    totalArea = layouts.reduce(
      (sum, item) =>
        sum + (item.width + componentGap) * (item.height + componentGap),
      0,
    ),
    targetWidth = Math.max(
      width - margin * 2,
      Math.sqrt(Math.max(1, totalArea)) * 1.45,
    ),
    drawn = new Map();
  let cursorX = margin,
    cursorY = margin,
    rowHeight = 0,
    graphWidth = 0,
    graphHeight = 0;
  layouts.forEach((layout) => {
    if (cursorX > margin && cursorX + layout.width > targetWidth + margin) {
      cursorX = margin;
      cursorY += rowHeight + componentGap;
      rowHeight = 0;
    }
    layout.drawn.forEach((node, id) =>
      drawn.set(id, { ...node, x: node.x + cursorX, y: node.y + cursorY }),
    );
    cursorX += layout.width + componentGap;
    rowHeight = Math.max(rowHeight, layout.height);
    graphWidth = Math.max(graphWidth, cursorX - componentGap + margin);
    graphHeight = Math.max(graphHeight, cursorY + layout.height + margin);
  });
  if (isolates.length) {
    const grid = topologyLayoutIsolates(
        isolates.sort((a, b) => a.id.localeCompare(b.id)),
        nodeW,
        nodeH,
      ),
      placeRight = graphHeight > graphWidth && layouts.length > 0,
      offsetX = placeRight ? graphWidth + componentGap : margin,
      offsetY = placeRight
        ? margin
        : Math.max(margin, graphHeight + componentGap);
    grid.drawn.forEach((node, id) =>
      drawn.set(id, { ...node, x: node.x + offsetX, y: node.y + offsetY }),
    );
    graphWidth = Math.max(graphWidth, offsetX + grid.width + margin);
    graphHeight = Math.max(graphHeight, offsetY + grid.height + margin);
  }
  if (!layouts.length && !isolates.length) return drawn;
  drawn.layoutBounds = {
    width: Math.max(width, graphWidth),
    height: Math.max(560, graphHeight),
  };
  return drawn;
}

export function topologyStraightEdges(edges, drawn) {
  const radius = 27,
    segments = new Map();
  [...edges]
    .sort((a, b) => a.id.localeCompare(b.id))
    .forEach((edge) => {
      const A = drawn.get(edge.source),
        B = drawn.get(edge.target);
      if (!A || !B || edge.source === edge.target) return;
      const ax = A.x + A.w / 2,
        ay = A.y + 30,
        bx = B.x + B.w / 2,
        by = B.y + 30,
        dx = bx - ax,
        dy = by - ay,
        length = Math.hypot(dx, dy);
      if (!length) return;
      const ux = dx / length,
        uy = dy / length,
        x1 = ax + ux * radius,
        y1 = ay + uy * radius,
        x2 = bx - ux * radius,
        y2 = by - uy * radius;
      segments.set(edge.id, {
        x1,
        y1,
        x2,
        y2,
        bounds: {
          minX: Math.min(x1, x2),
          maxX: Math.max(x1, x2),
          minY: Math.min(y1, y2),
          maxY: Math.max(y1, y2),
        },
      });
    });
  return segments;
}

export function topologyFocusLayout(
  drawn,
  selectedId,
  edges,
  width,
  height,
  nodeW,
  nodeH,
) {
  if (!selectedId || !drawn.has(selectedId)) return;
  const neighborIDs = new Set();
  edges.forEach((edge) => {
    if (edge.source === selectedId) neighborIDs.add(edge.target);
    if (edge.target === selectedId) neighborIDs.add(edge.source);
  });
  const neighbors = [...neighborIDs]
    .map((id) => drawn.get(id))
    .filter(Boolean)
    .sort((a, b) => {
      const layerOrder = {
          application: 0,
          workload: 1,
          runtime: 2,
          infrastructure: 3,
        },
        delta = layerOrder[topologyLayer(a)] - layerOrder[topologyLayer(b)];
      return delta || a.name.localeCompare(b.name) || a.id.localeCompare(b.id);
    });
  const cx = Math.max(nodeW / 2 + 32, width / 2),
    cy = Math.max(nodeH / 2 + 48, height / 2),
    rx = Math.min(310, Math.max(190, width * 0.29)),
    ry = Math.min(230, Math.max(150, height * 0.31));
  drawn.set(selectedId, {
    ...drawn.get(selectedId),
    x: cx - nodeW / 2,
    y: cy - nodeH / 2,
  });
  const layerOrder = {
      application: 0,
      workload: 1,
      runtime: 2,
      infrastructure: 3,
    },
    selectedLayer = layerOrder[topologyLayer(drawn.get(selectedId))],
    above = [],
    beside = [],
    below = [];
  neighbors.forEach((node) => {
    const layer = layerOrder[topologyLayer(node)];
    if (layer < selectedLayer) above.push(node);
    else if (layer > selectedLayer) below.push(node);
    else beside.push(node);
  });
  const placeRow = (items, y) =>
    items.forEach((node, index) => {
      const spread = Math.min(
          rx * 1.55,
          Math.max(0, (items.length - 1) * (nodeW + 26)),
        ),
        x =
          items.length === 1
            ? cx
            : cx - spread / 2 + (spread * index) / (items.length - 1);
      drawn.set(node.id, { ...node, x: x - nodeW / 2, y: y - nodeH / 2 });
    });
  placeRow(above, cy - ry);
  placeRow(below, cy + ry);
  beside.forEach((node, index) => {
    const side = index % 2 === 0 ? -1 : 1,
      level = Math.floor(index / 2),
      y = cy + (level - (Math.ceil(beside.length / 2) - 1) / 2) * (nodeH + 24);
    drawn.set(node.id, {
      ...node,
      x: cx + side * rx - nodeW / 2,
      y: y - nodeH / 2,
    });
  });
  const focused = new Set([selectedId, ...neighbors.map((node) => node.id)]),
    unrelated = [...drawn.values()]
      .filter((node) => !focused.has(node.id))
      .sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id));
  unrelated.forEach((node, index) =>
    drawn.set(node.id, {
      ...node,
      x: width + 80 + (index % 4) * (nodeW + 24),
      y: 40 + Math.floor(index / 4) * (nodeH + 22),
    }),
  );
}
