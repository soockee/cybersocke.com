(function(){
  function init(){
    const btn = document.querySelector('[data-nav-toggle]');
    const list = document.querySelector('[data-nav-list]');
    if(!btn || !list) return;

    function set(open){
      list.classList.toggle('open', open);
      btn.setAttribute('aria-expanded', String(open));
      document.body.classList.toggle('nav-open', open);
    }
    function auto(){
      if(window.innerWidth > 820){
        // desktop: ensure list visible without overlay semantics
        list.classList.remove('open');
        document.body.classList.remove('nav-open');
        btn.setAttribute('aria-expanded', 'false');
      } else {
        // mobile: keep hidden until user toggles
        list.classList.remove('open');
        document.body.classList.remove('nav-open');
        btn.setAttribute('aria-expanded', 'false');
      }
    }
    btn.addEventListener('click', function(){
      set(!list.classList.contains('open'));
    });
    window.addEventListener('resize', function(){
      auto();
    });
    document.addEventListener('keydown', function(e){
      if(e.key === 'Escape' && list.classList.contains('open') && window.innerWidth <= 820){
        set(false);
        btn.focus();
      }
    });
    // Close when clicking outside on mobile
    document.addEventListener('click', function(e){
      if(window.innerWidth > 820) return; // outside clicks only matter in mobile overlay mode
      if(!list.contains(e.target) && e.target !== btn && list.classList.contains('open')){
        set(false);
      }
    });
    auto();
  }
  document.readyState === 'loading' ? document.addEventListener('DOMContentLoaded', init) : init();
})();
