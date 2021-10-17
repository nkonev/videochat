export default (embedUploadFunction, chatId) => {
	/**
	 * Custom module for quilljs to allow user to drag images from their file system into the editor
	 * and paste images from clipboard (Works on Chrome, Firefox, Edge, not on Safari)
	 * @see https://quilljs.com/blog/building-a-custom-module/
	 */
	return class ImageDrop {
		imageRegex = /^image\/(gif|jpe?g|a?png|svg|webp|bmp|vnd\.microsoft\.icon)/i;

		/**
		 * Instantiate the module given a quill instance and any options
		 * @param {Quill} quill
		 * @param {Object} options
		 */
		constructor(quill, options = {}) {
			// save the quill reference
			this.quill = quill;
			// bind handlers to this instance
			this.handleDrop = this.handleDrop.bind(this);
			this.handlePaste = this.handlePaste.bind(this);
			// listen for drop and paste events
			this.quill.root.addEventListener('drop', this.handleDrop, false);
			this.quill.root.addEventListener('paste', this.handlePaste, false);
		}

		/**
		 * Handler for drop event to read dropped files from evt.dataTransfer
		 * @param {Event} evt
		 */
		handleDrop(evt) {
			evt.preventDefault();
			if (evt.dataTransfer && evt.dataTransfer.files && evt.dataTransfer.files.length) {
				if (document.caretRangeFromPoint) {
					const selection = document.getSelection();
					const range = document.caretRangeFromPoint(evt.clientX, evt.clientY);
					if (selection && range) {
						selection.setBaseAndExtent(range.startContainer, range.startOffset, range.startContainer, range.startOffset);
					}
				}
				this.readFiles(evt.dataTransfer.files, this.insert.bind(this));
			}
		}

		/**
		 * Handler for paste event to read pasted files from evt.clipboardData
		 * @param {Event} evt
		 */
		handlePaste(evt) {
			if (evt.clipboardData && evt.clipboardData.items && evt.clipboardData.items.length) {
				// console.log("evt", evt);
				if (evt.clipboardData.items[0].type.match(this.imageRegex)) {
					evt.preventDefault();
					this.readFiles(evt.clipboardData.items, url => {
						setTimeout(() => this.insert(url), 0);
					});
				}
			}
		}
		/**
		 * Insert the image into the document at the current cursor position
		 * @param {String} url  The base64-encoded image URI
		 */
		insert(url) {
			const index = (this.quill.getSelection() || {}).index || this.quill.getLength();
			this.quill.insertEmbed(index, 'image', url, 'user');
		}

		/**
		 * Extract image URIs a list of files from evt.dataTransfer or evt.clipboardData
		 * @param {File[]} files  One or more File objects
		 * @param {Function} callback  A function to send each data URI to
		 */
		readFiles(files, callback) {
			[].forEach.call(files, file => {
				if (!file.type.match(this.imageRegex)) {
					return;
				}
				const blob = file.getAsFile ? file.getAsFile() : file;
				embedUploadFunction(chatId, blob).then(url => callback(url))
			});
		}
	}
}
