import { useRef, useCallback, useMemo, useEffect, useState } from 'react';
import ForceGraph2D from 'react-force-graph-2d';
import { forceX, forceY } from 'd3-force';
import { useTheme, Box, Typography, useMediaQuery } from '@mui/material';
import { GraphData, GraphNode, GraphEdge } from '../types/graph';

interface NetworkGraphProps {
  data: GraphData;
  onNodeClick: (node: GraphNode) => void;
  selectedCircle?: string;
  showRelationships: boolean;
  showActivities: boolean;
  showCircles: boolean;
  centeredNodeId?: string;
}

interface ForceGraphData {
  nodes: GraphNode[];
  links: GraphEdge[];
}

export default function NetworkGraph({
  data,
  onNodeClick,
  selectedCircle,
  showRelationships,
  showActivities,
  showCircles,
  centeredNodeId,
}: NetworkGraphProps) {
  const theme = useTheme();
  const containerRef = useRef<HTMLDivElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const graphRef = useRef<any>(null);
  const [dimensions, setDimensions] = useState({ width: 800, height: 600 });
  const [hoveredEdge, setHoveredEdge] = useState<GraphEdge | null>(null);
  const [hoveredNode, setHoveredNode] = useState<GraphNode | null>(null);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));

  // Colors from theme
  const relationshipColor = theme.palette.primary.main;
  const activityColor = theme.palette.secondary.main;
  const circleEdgeColor = theme.palette.warning.main;
  const nodeColor = theme.palette.primary.main;
  const activityNodeColor = theme.palette.secondary.main;
  const circleNodeColor = theme.palette.warning.main;
  const textColor = theme.palette.text.primary;
  const bgColor = theme.palette.background.paper;

  // Handle container resize and mouse tracking
  useEffect(() => {
    const updateDimensions = () => {
      if (containerRef.current) {
        const { width, height } = containerRef.current.getBoundingClientRect();
        setDimensions({ width, height });
      }
    };

    const handleMouseMove = (e: MouseEvent) => {
      setTooltipPos({ x: e.clientX, y: e.clientY });
    };

    updateDimensions();
    window.addEventListener('resize', updateDimensions);
    window.addEventListener('mousemove', handleMouseMove);
    return () => {
      window.removeEventListener('resize', updateDimensions);
      window.removeEventListener('mousemove', handleMouseMove);
    };
  }, []);

  // Filter and transform data for the graph, including synthetic circle nodes/edges
  const graphData: ForceGraphData = useMemo(() => {
    let filteredNodes = data.nodes;

    // Filter by circle if selected
    if (selectedCircle) {
      const contactsInCircle = new Set(
        data.nodes
          .filter(n => n.type === 'contact' && n.circles?.includes(selectedCircle))
          .map(n => n.id)
      );

      // Include contacts in circle and activities that have at least 2 contacts in the circle
      filteredNodes = data.nodes.filter(n => {
        if (n.type === 'contact') {
          return contactsInCircle.has(n.id);
        }
        // For activities, check if they connect contacts in this circle
        const activityEdges = data.edges.filter(
          e => e.type === 'activity' &&
          (typeof e.source === 'string' ? e.source : e.source.id) === n.id
        );
        const connectedContacts = activityEdges.filter(e => {
          const targetId = typeof e.target === 'string' ? e.target : e.target.id;
          return contactsInCircle.has(targetId);
        });
        return connectedContacts.length >= 2;
      });
    }

    // Hide activity nodes when the activities toggle is off
    if (!showActivities) {
      filteredNodes = filteredNodes.filter(n => n.type !== 'activity');
    }

    const nodeIds = new Set(filteredNodes.map(n => n.id));

    // Filter edges based on visibility toggles and filtered nodes
    let filteredEdges = data.edges.filter(e => {
      const sourceId = typeof e.source === 'string' ? e.source : e.source.id;
      const targetId = typeof e.target === 'string' ? e.target : e.target.id;

      if (!nodeIds.has(sourceId) || !nodeIds.has(targetId)) return false;
      if (e.type === 'relationship' && !showRelationships) return false;
      if (e.type === 'activity' && !showActivities) return false;
      return true;
    });

    // Synthesize circle nodes and edges from contact circles data
    if (showCircles) {
      const visibleContacts = filteredNodes.filter(n => n.type === 'contact');

      // Count contacts per circle
      const circleContactMap = new Map<string, string[]>();
      visibleContacts.forEach(contact => {
        contact.circles?.forEach(circleName => {
          const existing = circleContactMap.get(circleName) ?? [];
          existing.push(contact.id);
          circleContactMap.set(circleName, existing);
        });
      });

      const circleNodes: GraphNode[] = [];
      const circleEdges: GraphEdge[] = [];

      circleContactMap.forEach((contactIds, circleName) => {
        if (contactIds.length < 2) return; // only show circles that connect people

        const circleNodeId = `circle-${circleName}`;
        circleNodes.push({
          id: circleNodeId,
          type: 'circle',
          label: circleName,
        });

        contactIds.forEach(contactId => {
          circleEdges.push({
            id: `ce-${contactId}-${circleName}`,
            type: 'circle',
            source: contactId,
            target: circleNodeId,
            label: circleName,
          });
        });
      });

      filteredNodes = [...filteredNodes, ...circleNodes];
      filteredEdges = [...filteredEdges, ...circleEdges];
    }

    return {
      nodes: filteredNodes,
      links: filteredEdges,
    };
  }, [data, selectedCircle, showRelationships, showActivities, showCircles]);

  // Center and zoom to selected node when centeredNodeId changes
  useEffect(() => {
    if (!centeredNodeId || !graphRef.current) return;

    const node = graphData.nodes.find(n => n.id === centeredNodeId);
    if (!node || node.x == null || node.y == null) return;

    graphRef.current.centerAt(node.x, node.y, 800);
    graphRef.current.zoom(2.5, 800);
  }, [centeredNodeId, graphData.nodes]);

  // Get initials from a name
  const getInitials = (label: string): string => {
    const parts = label.split(' ').filter(Boolean);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
    }
    return label.substring(0, 2).toUpperCase();
  };

  // Custom node rendering
  const nodeCanvasObject = useCallback((node: GraphNode, ctx: CanvasRenderingContext2D, globalScale: number) => {
    const isContact = node.type === 'contact';
    const isActivity = node.type === 'activity';
    const isCircleNode = node.type === 'circle';
    const size = isContact ? 12 : isCircleNode ? 9 : 6;
    const fontSize = Math.max(10 / globalScale, 3);
    const isCentered = node.id === centeredNodeId;

    // Draw highlight ring for centered contact
    if (isCentered) {
      ctx.beginPath();
      ctx.arc(node.x || 0, node.y || 0, size + 5, 0, 2 * Math.PI);
      ctx.strokeStyle = theme.palette.primary.light;
      ctx.lineWidth = 3 / globalScale;
      ctx.stroke();
    }

    // Draw node circle
    ctx.beginPath();
    ctx.arc(node.x || 0, node.y || 0, size, 0, 2 * Math.PI);
    if (isContact) {
      ctx.fillStyle = nodeColor;
    } else if (isActivity) {
      ctx.fillStyle = activityNodeColor;
    } else {
      ctx.fillStyle = circleNodeColor;
    }
    ctx.fill();

    // Draw border
    ctx.strokeStyle = bgColor;
    ctx.lineWidth = 2 / globalScale;
    ctx.stroke();

    // Draw initials for contacts
    if (isContact && globalScale > 0.5) {
      ctx.font = `bold ${fontSize * 1.2}px Inter, sans-serif`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillStyle = '#FFFFFF';
      ctx.fillText(getInitials(node.label), node.x || 0, node.y || 0);
    }

    // Draw label below node when zoomed in enough
    if (globalScale > 0.6 && (isContact || isActivity || isCircleNode)) {
      ctx.font = `${fontSize}px Inter, sans-serif`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'top';
      ctx.fillStyle = textColor;
      ctx.fillText(node.label, node.x || 0, (node.y || 0) + size + 4);
    }
  }, [nodeColor, activityNodeColor, circleNodeColor, bgColor, textColor, centeredNodeId, theme.palette.primary.light]);

  // Custom link rendering
  const linkColor = useCallback((link: GraphEdge) => {
    if (link.type === 'relationship') return relationshipColor;
    if (link.type === 'activity') return activityColor;
    return circleEdgeColor;
  }, [relationshipColor, activityColor, circleEdgeColor]);

  // Handle node hover
  const handleNodeHover = useCallback((node: GraphNode | null) => {
    setHoveredNode(node);
  }, []);

  // Handle link hover
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleLinkHover = useCallback((link: any) => {
    setHoveredEdge(link as GraphEdge | null);
  }, []);

  // Handle node click
  const handleNodeClick = useCallback((node: GraphNode) => {
    if (node.type === 'contact') {
      onNodeClick(node);
    }
  }, [onNodeClick]);

  // Configure forces to prevent isolated nodes from drifting too far
  useEffect(() => {
    if (graphRef.current) {
      const fg = graphRef.current;
      fg.d3Force('x', forceX(0).strength(0.05));
      fg.d3Force('y', forceY(0).strength(0.05));
      fg.d3Force('charge')?.strength(-100);
    }
  }, []);

  // Zoom to fit on initial load
  useEffect(() => {
    if (graphRef.current && graphData.nodes.length > 0) {
      setTimeout(() => {
        graphRef.current?.zoomToFit(400, isMobile ? 50 : 80);
      }, 500);
    }
  }, [graphData.nodes.length, isMobile]);

  const getEdgeTypeLabel = (type: string) => {
    if (type === 'relationship') return 'Relationship';
    if (type === 'activity') return 'Shared Activity';
    return 'Circle';
  };

  return (
    <Box ref={containerRef} sx={{ width: '100%', height: '100%', position: 'relative' }}>
      <ForceGraph2D
        ref={graphRef}
        width={dimensions.width}
        height={dimensions.height}
        graphData={graphData}
        nodeCanvasObject={nodeCanvasObject}
        nodePointerAreaPaint={(node: GraphNode, color, ctx) => {
          const size = node.type === 'contact' ? 12 : node.type === 'circle' ? 9 : 6;
          ctx.beginPath();
          ctx.arc(node.x || 0, node.y || 0, size + 4, 0, 2 * Math.PI);
          ctx.fillStyle = color;
          ctx.fill();
        }}
        linkColor={linkColor}
        linkWidth={2}
        linkDirectionalArrowLength={0}
        onNodeClick={handleNodeClick}
        onNodeHover={handleNodeHover}
        onLinkHover={handleLinkHover}
        cooldownTicks={100}
        enableNodeDrag={true}
        enableZoomInteraction={true}
        enablePanInteraction={true}
        backgroundColor={bgColor}
        nodeId="id"
        linkSource="source"
        linkTarget="target"
      />

      {/* Node / edge tooltip */}
      {(hoveredNode || hoveredEdge) && (
        <Box
          sx={{
            position: 'fixed',
            left: tooltipPos.x + 10,
            top: tooltipPos.y + 10,
            bgcolor: 'background.paper',
            border: 1,
            borderColor: 'divider',
            borderRadius: 1,
            px: 1.5,
            py: 0.75,
            boxShadow: 2,
            pointerEvents: 'none',
            zIndex: 1000,
          }}
        >
          {hoveredNode ? (
            <>
              <Typography variant="body2" sx={{ fontWeight: 500 }}>
                {hoveredNode.label}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {hoveredNode.type === 'contact' ? 'Contact' : hoveredNode.type === 'activity' ? 'Activity' : 'Circle'}
              </Typography>
            </>
          ) : hoveredEdge ? (
            <>
              <Typography variant="body2" sx={{ fontWeight: 500 }}>
                {hoveredEdge.label}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {getEdgeTypeLabel(hoveredEdge.type)}
              </Typography>
            </>
          ) : null}
        </Box>
      )}
    </Box>
  );
}
