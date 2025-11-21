// navigator.js: interactive note overlay navigation
(function(){
  const stack = [];
  function pushFragmentHTML(html){
    const wrapper = document.createElement('div');
    wrapper.className = 'overlay-layer';
    wrapper.innerHTML = html;
    document.body.appendChild(wrapper);
    stack.push(wrapper);
    bindFragment(wrapper);
    updateBodyState();
  }
  function openSlug(slug){
    fetch(`/posts/${slug}/fragment`).then(r=>r.text()).then(html=>{
      pushFragmentHTML(html);
    });
  }
  function openThemeBatch(tag){
    fetch(`/theme/fragments?tag=${encodeURIComponent(tag)}&limit=3`).then(r=>r.text()).then(html=>{
      const temp = document.createElement('div');
      temp.innerHTML = html;
      // each note-fragment child inside fragment-batch
      temp.querySelectorAll('.fragment-batch .note-fragment').forEach(frag=>{
        pushFragmentHTML(frag.outerHTML);
      });
    });
  }
  function closeTop(){
    const el = stack.pop();
    if(el){ el.remove(); }
    updateBodyState();
  }
  function bindFragment(root){
    root.querySelectorAll('.open-related').forEach(a=>{
      a.addEventListener('click', e=>{
        e.preventDefault();
        openSlug(a.dataset.slug);
      });
    });
    const closeBtn = root.querySelector('.close-fragment');
    if(closeBtn){ closeBtn.addEventListener('click', e=>{ e.preventDefault(); closeTop(); }); }
  }
  function updateBodyState(){
    document.body.dataset.overlayDepth = String(stack.length);
  }
  // Activation on post cards
  document.addEventListener('click', function(e){
    const card = e.target.closest('.postcard a');
    if(card && card.getAttribute('href').startsWith('/posts/')){
      const slug = card.getAttribute('href').replace('/posts/','');
      e.preventDefault();
      openSlug(slug);
      return;
    }
    const themeLink = e.target.closest('a.open-theme');
    if(themeLink){
      e.preventDefault();
      openThemeBatch(themeLink.dataset.themeTag);
    }
  });
})();
