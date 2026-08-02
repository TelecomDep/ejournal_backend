;/* === GFM Checkbox Enhancer v3 for Azimutt ===
 * Strategy: Use TreeWalker to find text nodes containing [x]/[ ] patterns
 * and replace them inline. This works WITH Elm's virtual DOM because:
 * - We only modify TEXT NODES, not element structure
 * - We wrap matched text in <span> which Elm treats as opaque
 * - We use requestAnimationFrame to run AFTER Elm's render cycle
 */
(function() {
  if (window.__gfmV3) return;
  window.__gfmV3 = true;

  // Inject minimal CSS for checkboxes
  var style = document.createElement('style');
  style.textContent = [
    '.gfm-cb { display: inline-block; width: 14px; height: 14px; border: 2px solid #d1d5db; border-radius: 3px; vertical-align: middle; margin-right: 4px; position: relative; background: white; }',
    '.gfm-cb-checked { background: #6366f1; border-color: #6366f1; }',
    '.gfm-cb-checked::after { content: ""; position: absolute; left: 3px; top: 0px; width: 4px; height: 8px; border: solid white; border-width: 0 2px 2px 0; transform: rotate(45deg); }',
    '.gfm-bullet-hide { list-style: none !important; }'
  ].join('\n');
  document.head.appendChild(style);

  console.log('🚀 GFM v3: Script loaded, waiting for DOM...');

  function makeCheckbox(checked) {
    var span = document.createElement('span');
    span.className = 'gfm-cb' + (checked ? ' gfm-cb-checked' : '');
    span.setAttribute('data-gfm', '1');
    return span;
  }

  function processNode() {
    // Walk ALL text nodes in document
    var walker = document.createTreeWalker(
      document.body,
      NodeFilter.SHOW_TEXT,
      null,
      false
    );

    var textNodes = [];
    var node;
    while (node = walker.nextNode()) {
      var text = node.nodeValue;
      if (text && (text.indexOf('[x]') !== -1 || text.indexOf('[X]') !== -1 || text.indexOf('[ ]') !== -1)) {
        // Don't process inside textareas or inputs
        var parent = node.parentNode;
        if (parent && (parent.tagName === 'TEXTAREA' || parent.tagName === 'INPUT' || parent.tagName === 'SCRIPT' || parent.tagName === 'STYLE')) continue;
        // Don't re-process
        if (parent && parent.getAttribute && parent.getAttribute('data-gfm-processed')) continue;
        textNodes.push(node);
      }
    }

    textNodes.forEach(function(textNode) {
      var text = textNode.nodeValue;
      var parent = textNode.parentNode;
      if (!parent) return;

      // Mark parent to avoid re-processing
      parent.setAttribute('data-gfm-processed', '1');

      // Split text around checkbox patterns
      var parts = text.split(/(\[[ xX]\])/);
      if (parts.length <= 1) return;

      var fragment = document.createDocumentFragment();
      parts.forEach(function(part) {
        if (part === '[x]' || part === '[X]') {
          fragment.appendChild(makeCheckbox(true));
        } else if (part === '[ ]') {
          fragment.appendChild(makeCheckbox(false));
        } else {
          // Remove leading "- " before checkbox (list marker)
          if (part.match(/^(.*)-\s*$/)) {
            var cleaned = part.replace(/-\s*$/, '');
            if (cleaned) fragment.appendChild(document.createTextNode(cleaned));
          } else {
            fragment.appendChild(document.createTextNode(part));
          }
        }
      });

      parent.replaceChild(fragment, textNode);

      // Hide bullet on parent li
      var li = parent.closest ? parent.closest('li') : null;
      if (!li && parent.tagName === 'LI') li = parent;
      if (li) {
        li.style.listStyleType = 'none';
        li.classList.add('gfm-bullet-hide');
      }
    });
  }

  function run() {
    try {
      processNode();
    } catch(e) {
      console.warn('GFM v3 error:', e);
    }
  }

  // Run after Elm renders using requestAnimationFrame
  function scheduleRun() {
    requestAnimationFrame(function() {
      requestAnimationFrame(run);
    });
  }

  // MutationObserver to detect Elm DOM updates
  var observer = new MutationObserver(function(mutations) {
    var dominated = false;
    for (var i = 0; i < mutations.length; i++) {
      if (mutations[i].addedNodes.length > 0 || mutations[i].type === 'characterData') {
        dominated = true;
        break;
      }
    }
    if (dominated) scheduleRun();
  });

  function init() {
    console.log('🚀 GFM v3: Observer started on document.body');
    observer.observe(document.body, { childList: true, subtree: true, characterData: true });
    run();
    // Also poll every 500ms as fallback
    setInterval(run, 500);
  }

  if (document.body) {
    init();
  } else {
    document.addEventListener('DOMContentLoaded', init);
  }
})();
