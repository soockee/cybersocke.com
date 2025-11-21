// tag_note_graph.js: simplified bipartite tag↔note graph
(function(){
  function init(){
    const container = document.getElementById('tag-note-cy');
    if(!container || typeof cytoscape === 'undefined') return;
    const dataRoot = container.parentElement.querySelector('.tag-note-data');
    if(!dataRoot) return;

    const tagSpans = dataRoot.querySelectorAll('.tag-node');
    const noteSpans = dataRoot.querySelectorAll('.note-node');
    const elements = [];
    let maxWeight = 0;

    // Build tag nodes
    tagSpans.forEach(s => {
      const id = 'tag:' + s.dataset.tag;
      const weight = parseInt(s.dataset.weight || '0', 10);
      if(weight > maxWeight) maxWeight = weight;
      elements.push({ data: { id, label: s.dataset.tag, type: 'tag', weight } });
    });
    // Build note nodes & edges tag->note
    noteSpans.forEach(s => {
      const id = 'note:' + s.dataset.slug;
      const weight = parseInt(s.dataset.weight || '0', 10);
      if(weight > maxWeight) maxWeight = weight;
      elements.push({ data: { id, label: s.dataset.name, type: 'note', weight, slug: s.dataset.slug } });
      (s.dataset.tags || '').split(/\s+/).filter(Boolean).forEach(t => {
        elements.push({ data: { id: 'e:' + t + '->' + s.dataset.slug, source: 'tag:' + t, target: id } });
      });
    });

    const cy = cytoscape({
      container,
      elements,
      layout: {
        name: 'fcose',
        quality: 'proof',              // higher polish for potentially larger bipartite sets
        randomize: true,
        animate: true,
        animationDuration: 1000,
        animationEasing: undefined,
        fit: true,
        padding: 50,
  nodeDimensionsIncludeLabels: true, // allow fcose to consider label box to reduce overlaps
        uniformNodeDimensions: false,
        packComponents: true,          // allow packing disconnected tags/notes
        step: 'all',
        samplingType: true,
        sampleSize: 25,
        nodeSeparation: 75,
        piTol: 1e-7,
  nodeRepulsion: n => 6800,      // enhanced separation for label-aware sizing
  idealEdgeLength: e => 120,     // slightly longer edges for readability
        edgeElasticity: e => 0.5,
        nestingFactor: 0.9,
        numIter: 1400,
        tile: true,
        tilingPaddingVertical: 14,
        tilingPaddingHorizontal: 14,
        gravity: 0.3,
        gravityRangeCompound: 1.6,
        gravityCompound: 1.0,
        gravityRange: 3.8,
        initialEnergyOnIncremental: 0.5,
        fixedNodeConstraint: undefined,
        alignmentConstraint: undefined,
        relativePlacementConstraint: undefined,
        ready: () => {},
        stop: () => {}
      },
      style: [
        { selector: 'node', style: {
          'label': 'data(label)',
          'font-size': 11,
          'text-wrap': 'wrap',
          'text-max-width': 90,
          'color': '#000',
          'background-color': '#000',
          'border-color': '#000',
          'border-width': 1,
          'text-background-opacity': 0,
          'shape': 'ellipse',
          'width': ele => 28 + (ele.data('weight') / (maxWeight || 1)) * 42,
          'height': ele => 28 + (ele.data('weight') / (maxWeight || 1)) * 42
        }},
        { selector: 'node[type="tag"]', style: { 'background-color': '#16a34a' }},
        { selector: 'edge', style: { 'line-color': '#444', 'curve-style': 'straight', 'width': 1.5 }},
        { selector: 'node:selected', style: { 'border-width': 3, 'border-color': '#4a90e2' }}
      ]
    });

    // Navigation
    cy.on('tap', 'node', evt => {
      const n = evt.target;
      const type = n.data('type');
      if(type === 'note') {
        const slug = n.data('slug');
        if(slug) window.location.href = '/posts/' + slug;
      } else if(type === 'tag') {
        const tag = n.data('label');
        if(tag) window.location.href = '/?tags=' + encodeURIComponent(tag);
      }
    });

    cy.on('layoutstop', () => cy.fit(undefined, 40));
    cy.fit(undefined, 40);
  }
  document.readyState === 'loading' ? document.addEventListener('DOMContentLoaded', init) : init();
})();
