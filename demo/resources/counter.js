
(function() {

    // Find our container
    // Since this could be on the page multiple times we'll search for the last instance.
    const container = Array.from(document.querySelectorAll('[data-counter-view]')).pop();
    if (!container) return;

    // Get the initial count from the attribute value otherwise start at 0
    let count = Number(container.getAttribute('data-counter-view') || 0);

    // Increment each second

    const increment = () => {
        container.innerText = count;
        count++;
    };

    increment();
    setInterval(increment, 1000);
})();