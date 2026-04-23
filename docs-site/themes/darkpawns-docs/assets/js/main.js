// Dark Pawns Documentation JavaScript

document.addEventListener('DOMContentLoaded', function() {
    // Copy button functionality
    const copyButtons = document.querySelectorAll('.copy-button');
    copyButtons.forEach(button => {
        button.addEventListener('click', function() {
            const targetId = this.getAttribute('data-clipboard-target');
            const codeElement = document.querySelector(targetId);
            
            if (codeElement) {
                const text = codeElement.textContent;
                navigator.clipboard.writeText(text).then(() => {
                    const originalText = this.innerHTML;
                    this.innerHTML = '<i class="fas fa-check"></i> Copied!';
                    this.style.backgroundColor = '#48bb78';
                    
                    setTimeout(() => {
                        this.innerHTML = originalText;
                        this.style.backgroundColor = '';
                    }, 2000);
                });
            }
        });
    });
    
    // Search functionality
    const searchInput = document.getElementById('search-input');
    const searchResults = document.getElementById('search-results');
    
    if (searchInput) {
        let searchIndex = null;
        
        // Load search index
        fetch('/docs/search-index.json')
            .then(response => response.json())
            .then(data => {
                searchIndex = data;
            })
            .catch(error => {
                console.error('Error loading search index:', error);
            });
        
        searchInput.addEventListener('input', function() {
            const query = this.value.trim().toLowerCase();
            
            if (!query || !searchIndex) {
                searchResults.classList.remove('show');
                return;
            }
            
            const results = searchIndex.filter(item => {
                return item.title.toLowerCase().includes(query) || 
                       item.content.toLowerCase().includes(query) ||
                       item.tags.some(tag => tag.toLowerCase().includes(query));
            }).slice(0, 10);
            
            if (results.length > 0) {
                searchResults.innerHTML = results.map(result => `
                    <div class="search-result-item">
                        <a href="${result.url}">${result.title}</a>
                        <p>${result.description || result.content.substring(0, 150)}...</p>
                    </div>
                `).join('');
                searchResults.classList.add('show');
            } else {
                searchResults.innerHTML = '<div class="search-result-item"><p>No results found</p></div>';
                searchResults.classList.add('show');
            }
        });
        
        // Close search results when clicking outside
        document.addEventListener('click', function(event) {
            if (!searchInput.contains(event.target) && !searchResults.contains(event.target)) {
                searchResults.classList.remove('show');
            }
        });
    }
    
    // Syntax highlighting for dynamic content
    if (typeof hljs !== 'undefined') {
        document.querySelectorAll('pre code').forEach((block) => {
            hljs.highlightElement(block);
        });
    }
    
    // Smooth scrolling for anchor links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            e.preventDefault();
            const targetId = this.getAttribute('href');
            if (targetId === '#') return;
            
            const targetElement = document.querySelector(targetId);
            if (targetElement) {
                targetElement.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });
    
    // Mobile menu toggle (if needed in future)
    const mobileMenuToggle = document.querySelector('.navbar-burger');
    if (mobileMenuToggle) {
        mobileMenuToggle.addEventListener('click', function() {
            const target = this.getAttribute('data-target');
            const menu = document.getElementById(target);
            
            this.classList.toggle('is-active');
            menu.classList.toggle('is-active');
        });
    }
});