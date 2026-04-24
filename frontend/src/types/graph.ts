// Types for the network graph visualization

export interface GraphNode {
  id: string;           // "c-{contactID}" for contacts, "a-{activityID}" for activities, "circle-{name}" for circles
  type: 'contact' | 'activity' | 'circle';
  label: string;        // Contact name, activity title, or circle name
  photo_thumbnail?: string;
  circles?: string[];
  // Properties added by force-graph during rendering
  x?: number;
  y?: number;
  vx?: number;
  vy?: number;
}

export interface GraphEdge {
  id: string;
  source: string | GraphNode;  // Can be ID string or resolved node object
  target: string | GraphNode;
  type: 'relationship' | 'activity' | 'circle';
  label: string;
}

export interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

// Response from the API
export interface GraphResponse {
  nodes: GraphNode[];
  edges: GraphEdge[];
}
