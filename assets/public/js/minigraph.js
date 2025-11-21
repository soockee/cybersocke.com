// minigraph.js: minimal Cytoscape mini graph (focus node + adjacency edges)
(function(){
  function init(){
    const container = document.getElementById('mini-cy');
    if(!container || typeof cytoscape === 'undefined') return;

    // Collect adjacency rows (hidden markup produced by template)
    let rows = container.parentElement.querySelectorAll('.mini-cy-adjacency .adjacency-row');
    if(!rows.length) rows = document.querySelectorAll('.mini-cy-adjacency .adjacency-row');
    if(!rows.length) return;

    const elements = [];
    let maxNodeWeight = 0;
    let maxEdgeWeight = 0;

    // Nodes
    rows.forEach(r => {
      const id = r.dataset.slug;
      const label = r.dataset.name || id;
      const weight = parseInt(r.dataset.weight || '0', 10);
      if(weight > maxNodeWeight) maxNodeWeight = weight;
      elements.push({ data: { id, label, weight } });
    });

    // Focus node adjustments
    const focusId = rows[0].dataset.slug;
    const focusNode = elements.find(e => e.data.id === focusId);
    if(focusNode){
      focusNode.data.weight = maxNodeWeight + 1; // make focus a bit larger
      maxNodeWeight = focusNode.data.weight;
      const mode = container.dataset.mode || 'post';
      const focusLabel = container.dataset.focusLabel || '';
      focusNode.data.label = (mode === 'tag') ? focusLabel : '';
    }

    // Edges: focus -> others
    rows.forEach((r, idx) => {
      if(idx === 0) return;
      const target = r.dataset.slug;
      const weight = parseInt(r.dataset.weight || '0', 10);
      if(weight > maxEdgeWeight) maxEdgeWeight = weight;
      elements.push({ data: { id: focusId + '->' + target, source: focusId, target, weight } });
    });

    const cy = cytoscape({
      container,
      elements,
      layout: {
        name: 'fcose',
        quality: 'proof',              // higher quality to account for label dimensions
        randomize: true,
        animate: true,
        animationDuration: 800,
        fit: true,
        padding: 40,
        nodeDimensionsIncludeLabels: true, // include labels in collision/spacing
        uniformNodeDimensions: false,
        packComponents: false,
        step: 'all',
        samplingType: true,
        sampleSize: 20,
        nodeSeparation: 70,
        piTol: 1e-7,
  nodeRepulsion: n => 6000,      // slightly stronger to offset larger label-aware nodes
  idealEdgeLength: e => 130,     // more spacing with larger effective node boxes
        edgeElasticity: e => 0.45,
        nestingFactor: 0.75,
        numIter: 900,
        tile: false,
        tilingPaddingVertical: 8,
        tilingPaddingHorizontal: 8,
        gravity: 0.25,
        gravityRangeCompound: 1.5,
        gravityCompound: 1.0,
        gravityRange: 3.5,
        initialEnergyOnIncremental: 0.35,
        ready: () => {},
        stop: () => {}
      },
      style: [
        { selector: 'node', style: {
          'label': 'data(label)',
          'font-size': 11,
          'text-wrap': 'wrap',
          'text-max-width': 80,
          'text-valign': 'bottom', // regular nodes: place label below node center
          'text-halign': 'center',
          'text-margin-y': 6, // push label slightly outward
          'background-color': '#fff',
          'border-color': '#000',
          'border-width': 1,
          'color': '#000',
          'text-background-opacity': 0,
          'text-background-padding': '2px',
          'shape': 'ellipse',
          'width': ele => 38 + (ele.data('weight') / (maxNodeWeight || 1)) * 34,
          'height': ele => 24 + (ele.data('weight') / (maxNodeWeight || 1)) * 18
        }},
        { selector: 'node.focus', style: {
          'background-color': '#16a34a',
          'color': '#000000',
          'border-width': 2,
          'text-valign': 'center', // focus node: keep label inside the shape
          'text-halign': 'center',
          'text-margin-y': 0,
          'width': ele => {
            const w = ele.data('weight');
            const label = ele.data('label') || '';
            const base = 38 + (w / (maxNodeWeight || 1)) * 34;
            const needed = 24 + label.length * 6;
            return Math.max(base, Math.min(needed, 110));
          },
          'height': ele => {
            const w = ele.data('weight');
            const label = ele.data('label') || '';
            const base = 24 + (w / (maxNodeWeight || 1)) * 18;
            const lines = Math.ceil(label.length / 12);
            return Math.max(base, 24 + (lines - 1) * 12);
          }
        }},
        { selector: 'edge', style: {
          'line-color': '#000',
          'curve-style': 'straight',
          'width': ele => 1 + (ele.data('weight') / (maxEdgeWeight || 1)) * 3
        }}
      ]
    });

    cy.getElementById(focusId).addClass('focus');
    cy.on('layoutstop', () => cy.fit(undefined, 30));
    cy.fit(undefined, 30);

    // Simple navigation (skip focus)
    cy.on('tap', 'node', evt => {
      const id = evt.target.id();
      if(id !== focusId) window.location.href = '/posts/' + id;
    });
  }

  document.readyState === 'loading' ? document.addEventListener('DOMContentLoaded', init) : init();
})();
