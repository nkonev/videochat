import {Image} from '@tiptap/extension-image';
import {Plugin, PluginKey} from 'prosemirror-state';
import {hasLength} from "@/utils";

export const buildImageHandler = (uploadFunction) => {
    return Image.extend({
        addProseMirrorPlugins() {
            return [
                new Plugin({
                    key: new PluginKey('imageHandler'),
                    props: {

                        handleDOMEvents: {
                            drop: (view, event) => {
                                const hasFiles =
                                    event.dataTransfer &&
                                    event.dataTransfer.files &&
                                    event.dataTransfer.files.length;

                                if (!hasFiles) {
                                    return false;
                                }

                                const images = Array.from(
                                    event.dataTransfer?.files ?? []
                                ).filter((file) => /image/i.test(file.type));

                                if (images.length === 0) {
                                    return false;
                                }

                                event.preventDefault();

                                const { schema } = view.state;
                                const coordinates = view.posAtCoords({
                                    left: event.clientX,
                                    top: event.clientY,
                                });
                                if (!coordinates) return false;

                                images.forEach(async (image) => {
                                    await uploadFunction(image);
                                });

                                return true;
                            },
                            // we place handler here (but not in props)
                            // in order to pass text if no image was found
                            paste(view, event) {
                                  let imageSet = false;
                                  const items = (event.clipboardData || event.originalEvent.clipboardData).items;
                                  for (const item of items) {
                                      if (item.type.indexOf("image") === 0) {
                                          imageSet = true;

                                          event.preventDefault();

                                          const image = item.getAsFile();
                                          uploadFunction(image, true);
                                      }
                                  }
                                  if (imageSet) {
                                      return true
                                  } else {
                                      return false
                                  }
                            }
                        },
                    }
                }),
            ];
        }
    })
};
