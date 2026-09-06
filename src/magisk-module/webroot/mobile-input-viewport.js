(() => {
  const editableSelector = 'input, textarea, select, [contenteditable="true"]';
  const viewport = window.visualViewport || null;
  let activeControl = null;
  let settleTimer = 0;

  function isEditableControl(node) {
    return node instanceof Element && node.matches(editableSelector);
  }

  function keepActiveControlVisible() {
    if (!activeControl || !document.contains(activeControl)) return;
    const rect = activeControl.getBoundingClientRect();
    const top = viewport ? viewport.offsetTop : 0;
    const height = viewport ? viewport.height : window.innerHeight;
    const bottom = top + height;
    const margin = Math.min(96, Math.max(24, height * 0.12));
    if (rect.bottom > bottom - margin || rect.top < top + margin) {
      activeControl.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'auto' });
    }
  }

  function scheduleVisibilityCheck(delay = 120) {
    window.clearTimeout(settleTimer);
    settleTimer = window.setTimeout(keepActiveControlVisible, delay);
  }

  document.addEventListener('focusin', (event) => {
    if (!isEditableControl(event.target)) return;
    activeControl = event.target;
    scheduleVisibilityCheck(80);
    window.setTimeout(keepActiveControlVisible, 320);
  }, true);

  document.addEventListener('focusout', (event) => {
    if (event.target === activeControl) activeControl = null;
  }, true);

  if (viewport) {
    viewport.addEventListener('resize', () => scheduleVisibilityCheck(40));
    viewport.addEventListener('scroll', () => scheduleVisibilityCheck(40));
  }
})();
