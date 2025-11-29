import { Extension } from '@tiptap/core';
import { Plugin } from 'prosemirror-state';

// Custom extension for paste handling
export default Extension.create({
    name: 'cleanPaste',

    addProseMirrorPlugins() {
        return [
            new Plugin({
                props: {
                    handlePaste: () => false,
                    transformPasted: (slice) => {
                        // Let the main HTML transformer do the heavy lifting
                        return slice;
                    }
                }
            })
        ];
    }
});



export function cleanPastedHTML(html) {
    try {
        // Create a document fragment
        const tempContainer = document.createElement('div');
        tempContainer.innerHTML = html;

        // Remove all style attributes
        const elementsWithStyle = tempContainer.querySelectorAll('*[style]');
        elementsWithStyle.forEach(el => el.removeAttribute('style'));

        // Remove all class attributes
        const elementsWithClass = tempContainer.querySelectorAll('*[class]');
        elementsWithClass.forEach(el => el.removeAttribute('class'));

        // Remove data attributes (often used for hidden content)
        const elementsWithDataAttrs  = tempContainer.querySelectorAll('*');
        elementsWithDataAttrs.forEach(el => {
            Array.from(el.attributes)
                .filter(attr => attr.name.startsWith('data-'))
                .forEach(attr => el.removeAttribute(attr.name));
        });

        // Process empty wrappers
        removeEmptyWrappers(tempContainer);

        return tempContainer.innerHTML;
    } catch (error) {
        console.error('Error cleaning pasted HTML:', error);
        return html; // Fallback to original
    }
}

function removeEmptyWrappers(element) {
    // Get all children (as array to avoid live NodeList issues)
    const children = Array.from(element.children);

    // Process each child recursively first
    children.forEach(child => removeEmptyWrappers(child));

    // Then check each direct child
    children.forEach(child => {
        if ((child.tagName === 'SPAN' || child.tagName === 'DIV') &&
            (!child.attributes.length || hasOnlyEmptyAttributes(child))) {

            // Move child nodes before removing the element
            while (child.firstChild) {
                element.insertBefore(child.firstChild, child);
            }

            // Remove the now-empty element
            element.removeChild(child);
        }
    });
}

function hasOnlyEmptyAttributes(child) {
    let hasValuable = false;
    for (const attr of child.attributes) {
        if (attr.name.length > 0 || attr.value.length > 0) {
            hasValuable = true;
            break
        }
    }

    return !hasValuable
}
