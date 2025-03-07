<template>
  <div
    class="profile-picture"
    @mouseenter="hovered = true"
    @mouseleave="hovered = false"
    @click="openFileSelector"
  >
    <!-- Display profile picture -->
    <ProfilePicture :contactId="contactId" :width="120" :height="120" />
    <v-icon v-if="hovered" class="profile-hover-icon">mdi-pencil-circle</v-icon>

    <!-- Hidden file input -->
    <input
      type="file"
      accept="image/*"
      ref="fileInput"
      @change="onFileSelected"
      style="display: none"
    />

    <!-- Vuetify Dialog for cropping -->
    <v-dialog v-model="showCropModal" max-width="600px" persistent>
      <template v-slot:default>
        <v-card>
          <v-card-title class="headline">
            {{ $t("contacts.photo.crop_image") }}
          </v-card-title>
          <v-card-text>
            <canvas
              ref="canvas"
              class="crop-canvas"
              @mousedown="onMouseDown"
              @mousemove="onMouseMove"
              @mouseup="onMouseUp"
              @mouseleave="onMouseLeave"
            ></canvas>
          </v-card-text>
          <v-card-actions>
            <v-spacer></v-spacer>
            <v-btn color="primary" @click="cropImage">
              {{ $t("contacts.photo.crop") }}
            </v-btn>
            <v-btn color="secondary" @click="closeCropModal">
              {{ $t("buttons.cancel") }}
            </v-btn>
          </v-card-actions>
        </v-card>
      </template>
    </v-dialog>
  </div>
</template>

<script>
import contactService from "@/services/contactService";
import { backendURL } from "@/services/api";
import ProfilePicture from "@/components/ProfilePicture.vue";

export default {
  props: {
    contactId: {
      required: true,
    },
  },
  components: {
    ProfilePicture,
  },
  data() {
    return {
      defaultPicture: "/placeholder-avatar.png",
      showCropModal: false,
      selectedImage: null,
      canvasContext: null,
      imageElement: null,
      cropBox: { x: 100, y: 100, size: 150 }, // Initial crop box
      isDragging: false,
      isResizing: false,
      dragOffset: { x: 0, y: 0 },
      backendURL,
      hovered: false,
      reloadPicture: false,
    };
  },
  methods: {
    openFileSelector() {
      this.$refs.fileInput.click();
    },
    onFileSelected(event) {
      const file = event.target.files[0];
      if (file && file.type.startsWith("image/")) {
        const reader = new FileReader();
        reader.onload = (e) => {
          this.selectedImage = e.target.result;
          this.openCropModal();
        };
        reader.readAsDataURL(file);
      } else {
        alert("Please select a valid image file.");
      }
    },
    openCropModal() {
      this.showCropModal = true;
      this.$nextTick(() => {
        this.initializeCanvas();
      });
    },
    initializeCanvas() {
      const canvas = this.$refs.canvas;
      this.canvasContext = canvas.getContext("2d");

      this.imageElement = new Image();
      this.imageElement.onload = () => {
        // Compute the scaling factor to fit the image into the canvas
        const maxCanvasWidth = 500;
        const scale = Math.min(maxCanvasWidth / this.imageElement.width, 1);
        canvas.width = this.imageElement.width * scale;
        canvas.height = this.imageElement.height * scale;

        // Update crop box dimensions based on scaled canvas
        this.cropBox = {
          x: (canvas.width - 150) / 2,
          y: (canvas.height - 150) / 2,
          size: 150,
          aspectRatio: 1,
        };

        this.drawCanvas();
      };
      this.imageElement.src = this.selectedImage;
    },
    drawCanvas() {
      const canvas = this.$refs.canvas;
      const ctx = this.canvasContext;
      const { x, y, size } = this.cropBox;

      // Clear the canvas
      ctx.clearRect(0, 0, canvas.width, canvas.height);

      // Draw the scaled image to fit the canvas
      ctx.drawImage(this.imageElement, 0, 0, canvas.width, canvas.height);

      // Dim the area outside the crop box
      ctx.fillStyle = "rgba(0, 0, 0, 0.5)";
      ctx.fillRect(0, 0, canvas.width, canvas.height);

      // Map crop box dimensions to the original image
      const scaleX = this.imageElement.width / canvas.width;
      const scaleY = this.imageElement.height / canvas.height;

      const sx = x * scaleX;
      const sy = y * scaleY;
      const sWidth = size * scaleX;
      const sHeight = size * scaleY;

      // Clear the crop box area to reveal the image underneath
      ctx.clearRect(x, y, size, size);

      // Redraw the image inside the crop box
      ctx.drawImage(
        this.imageElement,
        sx,
        sy,
        sWidth,
        sHeight,
        x,
        y,
        size,
        size
      );

      // Draw the crop box border
      ctx.strokeStyle = "black";
      ctx.lineWidth = 2;
      ctx.strokeRect(x, y, size, size);

      // Draw the resize handle
      const handleSize = 10;
      ctx.fillStyle = "white";
      ctx.fillRect(
        x + size - handleSize / 2,
        y + size - handleSize / 2,
        handleSize,
        handleSize
      );
      ctx.strokeStyle = "black";
      ctx.strokeRect(
        x + size - handleSize / 2,
        y + size - handleSize / 2,
        handleSize,
        handleSize
      );
    },
    async cropImage() {
      const { x, y, size } = this.cropBox;
      const scale = this.imageElement.width / this.$refs.canvas.width;

      // Map crop box dimensions to the original image
      const sx = x * scale;
      const sy = y * scale;
      const sWidth = size * scale;
      const sHeight = size * scale;

      // Create a new canvas for the cropped image
      const outputCanvas = document.createElement("canvas");
      outputCanvas.width = size;
      outputCanvas.height = size;
      const outputContext = outputCanvas.getContext("2d");

      // Draw cropped area from the original image
      outputContext.drawImage(
        this.imageElement,
        sx,
        sy,
        sWidth,
        sHeight,
        0,
        0,
        size,
        size
      );

      // Convert canvas to Blob
      outputCanvas.toBlob(async (blob) => {
        try {
          const contactId = this.$props.contactId; // Assuming `contactId` is passed as a prop
          if (!contactId) throw new Error("Contact ID is not defined.");

          await this.handleUploadProfilePicture(contactId, blob);
          this.closeCropModal();
        } catch (error) {
          console.error("Failed to upload cropped profile picture:", error);
        }
      }, "image/png");
    },
    async handleUploadProfilePicture(contactId, photoFile) {
      try {
        await contactService.addPhotoToContact(contactId, photoFile);
        this.reloadPicture = Date.now();
        // Update the profile picture URL
        this.$emit("photoUploaded");
      } catch (error) {
        console.error("Failed to upload profile picture:", error);
      }
    },
    closeCropModal() {
      this.showCropModal = false;
      this.selectedImage = null;
    },
    onMouseDown(event) {
      const { offsetX, offsetY } = event;
      const { x, y, size } = this.cropBox;
      const resizeMargin = 10;

      // Check if resizing (within the resize handle)
      if (
        offsetX > x + size - resizeMargin &&
        offsetY > y + size - resizeMargin
      ) {
        this.isResizing = true;
      } else if (
        offsetX > x &&
        offsetX < x + size &&
        offsetY > y &&
        offsetY < y + size
      ) {
        // Check if dragging the crop box
        this.isDragging = true;
        this.dragOffset = { x: offsetX - x, y: offsetY - y };
      }
    },
    onMouseMove(event) {
      if (!this.isDragging && !this.isResizing) return;

      const canvas = this.$refs.canvas;
      const { offsetX, offsetY } = event;

      if (this.isDragging) {
        this.cropBox.x = Math.max(
          0,
          Math.min(
            offsetX - this.dragOffset.x,
            canvas.width - this.cropBox.size
          )
        );
        this.cropBox.y = Math.max(
          0,
          Math.min(
            offsetY - this.dragOffset.y,
            canvas.height - this.cropBox.size
          )
        );
      }

      if (this.isResizing) {
        const newSize = Math.min(
          canvas.width - this.cropBox.x,
          canvas.height - this.cropBox.y,
          offsetX - this.cropBox.x,
          offsetY - this.cropBox.y
        );
        this.cropBox.size = Math.max(50, newSize);
      }

      this.drawCanvas();
    },
    onMouseUp() {
      this.isDragging = false;
      this.isResizing = false;
    },
    onMouseLeave() {
      this.isDragging = false;
      this.isResizing = false;
    },
  },
};
</script>

<style scoped>
.profile-picture {
  position: relative;
  margin: auto;
  cursor: pointer;
}

.profile-hover-icon {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: white;
  color: black;
  border-radius: 50%;
  transition: opacity 0.3s ease;
}

.profile-picture:hover .profile-hover-icon {
  opacity: 1;
}

.crop-canvas {
  border-radius: 8px;
  margin: 16px 0;
  box-shadow: 0px 4px 8px rgba(0, 0, 0, 0.2);
  cursor: move;
}

.v-card-actions {
  display: flex;
  justify-content: flex-end;
}

.resize-handle {
  cursor: nwse-resize;
  background: white;
  border: 1px solid black;
}
</style>
