const DEFAULT_OUTPUT_SIZE = 320;
const DEFAULT_EXPANSION = 1.9;

export function drawFaceCropToCanvas(canvas, video, box, options = {}) {
  const sourceWidth = video?.videoWidth || 0;
  const sourceHeight = video?.videoHeight || 0;
  if (!canvas || !video || !sourceWidth || !sourceHeight) {
    throw new Error("Camera frame is not ready");
  }

  const outputSize = options.outputSize || DEFAULT_OUTPUT_SIZE;
  const crop = expandedSquareCrop(box, sourceWidth, sourceHeight, options.expansion || DEFAULT_EXPANSION);
  canvas.width = outputSize;
  canvas.height = outputSize;

  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Camera canvas is not available");
  }

  context.imageSmoothingEnabled = true;
  context.imageSmoothingQuality = "high";
  context.drawImage(video, crop.x, crop.y, crop.size, crop.size, 0, 0, outputSize, outputSize);
  return { width: outputSize, height: outputSize, crop };
}

export function drawFullFrameToCanvas(canvas, video, maxWidth = 640) {
  const sourceWidth = video?.videoWidth || 0;
  const sourceHeight = video?.videoHeight || 0;
  if (!canvas || !video || !sourceWidth || !sourceHeight) {
    throw new Error("Camera frame is not ready");
  }

  const width = Math.min(sourceWidth, maxWidth);
  const height = Math.round(width * (sourceHeight / sourceWidth));
  canvas.width = width;
  canvas.height = height;

  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Camera canvas is not available");
  }

  context.imageSmoothingEnabled = true;
  context.imageSmoothingQuality = "high";
  context.drawImage(video, 0, 0, width, height);
  return { width, height };
}

export function mapNormalizedBoxToElementStyle(box, video) {
  const element = video?.parentElement;
  const rect = element?.getBoundingClientRect();
  const sourceWidth = video?.videoWidth || 0;
  const sourceHeight = video?.videoHeight || 0;
  if (!box || !rect?.width || !rect?.height || !sourceWidth || !sourceHeight) {
    return undefined;
  }

  const sourceAspect = sourceWidth / sourceHeight;
  const elementAspect = rect.width / rect.height;
  let renderedWidth = rect.width;
  let renderedHeight = rect.height;
  let offsetX = 0;
  let offsetY = 0;

  if (sourceAspect > elementAspect) {
    renderedHeight = rect.width / sourceAspect;
    offsetY = (rect.height - renderedHeight) / 2;
  } else {
    renderedWidth = rect.height * sourceAspect;
    offsetX = (rect.width - renderedWidth) / 2;
  }

  const left = clamp01((offsetX + box.left * renderedWidth) / rect.width);
  const top = clamp01((offsetY + box.top * renderedHeight) / rect.height);
  const right = clamp01((offsetX + (box.left + box.width) * renderedWidth) / rect.width);
  const bottom = clamp01((offsetY + (box.top + box.height) * renderedHeight) / rect.height);

  return {
    left: `${left * 100}%`,
    top: `${top * 100}%`,
    width: `${Math.max(0.02, right - left) * 100}%`,
    height: `${Math.max(0.02, bottom - top) * 100}%`,
  };
}

export function mapFrameBoxToElementStyle(box, frameSize, video) {
  const element = video?.parentElement;
  const rect = element?.getBoundingClientRect();
  const frameWidth = frameSize?.width || video?.videoWidth || 0;
  const frameHeight = frameSize?.height || video?.videoHeight || 0;
  const crop = frameSize?.crop;
  const videoWidth = video?.videoWidth || 0;
  const videoHeight = video?.videoHeight || 0;
  if (!box || !rect?.width || !rect?.height || !frameWidth || !frameHeight) {
    return undefined;
  }

  const mappedBox = crop?.size && videoWidth && videoHeight
    ? {
        x: crop.x + (box.x / frameWidth) * crop.size,
        y: crop.y + (box.y / frameHeight) * crop.size,
        width: (box.width / frameWidth) * crop.size,
        height: (box.height / frameHeight) * crop.size,
      }
    : box;
  const sourceWidth = crop?.size && videoWidth ? videoWidth : frameWidth;
  const sourceHeight = crop?.size && videoHeight ? videoHeight : frameHeight;
  const sourceAspect = sourceWidth / sourceHeight;
  const elementAspect = rect.width / rect.height;
  let renderedWidth = rect.width;
  let renderedHeight = rect.height;
  let offsetX = 0;
  let offsetY = 0;

  if (sourceAspect > elementAspect) {
    renderedHeight = rect.width / sourceAspect;
    offsetY = (rect.height - renderedHeight) / 2;
  } else {
    renderedWidth = rect.height * sourceAspect;
    offsetX = (rect.width - renderedWidth) / 2;
  }

  const left = clamp01((offsetX + (mappedBox.x / sourceWidth) * renderedWidth) / rect.width);
  const top = clamp01((offsetY + (mappedBox.y / sourceHeight) * renderedHeight) / rect.height);
  const right = clamp01((offsetX + ((mappedBox.x + mappedBox.width) / sourceWidth) * renderedWidth) / rect.width);
  const bottom = clamp01((offsetY + ((mappedBox.y + mappedBox.height) / sourceHeight) * renderedHeight) / rect.height);

  return {
    left: `${left * 100}%`,
    top: `${top * 100}%`,
    width: `${Math.max(0.02, right - left) * 100}%`,
    height: `${Math.max(0.02, bottom - top) * 100}%`,
  };
}

function expandedSquareCrop(box, sourceWidth, sourceHeight, expansion) {
  if (!box) {
    return centeredCrop(sourceWidth, sourceHeight);
  }

  const faceLeft = clamp(box.left, 0, 1) * sourceWidth;
  const faceTop = clamp(box.top, 0, 1) * sourceHeight;
  const faceWidth = Math.max(1, clamp(box.width, 0, 1) * sourceWidth);
  const faceHeight = Math.max(1, clamp(box.height, 0, 1) * sourceHeight);
  const centerX = faceLeft + faceWidth / 2;
  const centerY = faceTop + faceHeight * 0.46;
  const maxSize = Math.min(sourceWidth, sourceHeight);
  const size = Math.min(maxSize, Math.max(faceWidth, faceHeight) * expansion);

  let x = centerX - size / 2;
  let y = centerY - size / 2;
  x = clamp(x, 0, sourceWidth - size);
  y = clamp(y, 0, sourceHeight - size);

  return {
    x: Math.round(x),
    y: Math.round(y),
    size: Math.max(1, Math.round(size)),
  };
}

function centeredCrop(sourceWidth, sourceHeight) {
  const size = Math.min(sourceWidth, sourceHeight);
  return {
    x: Math.round((sourceWidth - size) / 2),
    y: Math.round((sourceHeight - size) / 2),
    size,
  };
}

function clamp01(value) {
  return clamp(value, 0, 1);
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}
