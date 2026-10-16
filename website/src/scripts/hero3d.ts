// Runtime for the 3D hero: power flowing along every bay, GOOSE rings from the IED
// cabinets, a slow orbit with mouse parallax. Pauses off-screen and in hidden tabs.
import {
  AdditiveBlending, CanvasTexture, Color, DirectionalLight, Fog, HemisphereLight, Mesh, MeshBasicMaterial,
  PerspectiveCamera, Points, PointsMaterial, RingGeometry, Scene, SRGBColorSpace, Vector3, WebGLRenderer,
} from 'three';
import { buildSubstation, pointCloud } from './hero3d-scene';

const PER_FLOW = 9;

function glowTexture(): CanvasTexture {
  const canvas = document.createElement('canvas');
  canvas.width = canvas.height = 64;
  const context = canvas.getContext('2d')!;
  const gradient = context.createRadialGradient(32, 32, 0, 32, 32, 32);
  gradient.addColorStop(0, 'rgba(255,255,255,1)');
  gradient.addColorStop(0.25, 'rgba(255,255,255,0.75)');
  gradient.addColorStop(1, 'rgba(255,255,255,0)');
  context.fillStyle = gradient;
  context.fillRect(0, 0, 64, 64);
  const texture = new CanvasTexture(canvas);
  texture.colorSpace = SRGBColorSpace;
  return texture;
}

export function startHero(canvas: HTMLCanvasElement, onFirstFrame: () => void): () => void {
  const renderer = new WebGLRenderer({ canvas, antialias: true, alpha: true, powerPreference: 'high-performance' });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 1.75));
  renderer.setClearColor(0x000000, 0);
  const scene = new Scene();
  scene.fog = new Fog(0x070d1a, 24, 52);
  const camera = new PerspectiveCamera(34, 1, 0.1, 120);
  scene.add(new HemisphereLight(0xb8ccff, 0x0a1426, 1.15));
  const sun = new DirectionalLight(0xffffff, 1.9);
  sun.position.set(8, 14, 10);
  scene.add(sun);

  const { group, flows, cabinets } = buildSubstation();
  scene.add(group);
  const glow = glowTexture();
  const sparks = new Points(pointCloud(flows.length * PER_FLOW), new PointsMaterial({
    color: new Color(0x2be4b0), size: 0.62, map: glow, transparent: true, depthWrite: false, blending: AdditiveBlending,
  }));
  scene.add(sparks);
  const rings = cabinets.filter((_, i) => i % 2 === 0).map((position, i) => {
    const ring = new Mesh(new RingGeometry(0.9, 1.0, 48), new MeshBasicMaterial({
      color: 0x38bdf8, transparent: true, opacity: 0, depthWrite: false, blending: AdditiveBlending,
    }));
    ring.rotation.x = -Math.PI / 2;
    ring.position.copy(position);
    scene.add(ring);
    return { ring, phase: i * 0.9 };
  });

  const target = new Vector3(0, 1.6, 0);
  const pointer = { x: 0, y: 0, smoothX: 0, smoothY: 0 };
  let visible = true, running = true, frame = 0, first = true, last = performance.now(), angle = 0.6;

  const resize = () => {
    const width = canvas.clientWidth, height = canvas.clientHeight;
    if (!width || !height) return;
    renderer.setSize(width, height, false);
    camera.aspect = width / height;
    camera.updateProjectionMatrix();
  };
  const sizeObserver = new ResizeObserver(resize);
  sizeObserver.observe(canvas);
  const viewObserver = new IntersectionObserver(([entry]) => { visible = entry.isIntersecting; schedule(); });
  viewObserver.observe(canvas);
  const onPointer = (event: PointerEvent) => {
    pointer.x = event.clientX / window.innerWidth - 0.5;
    pointer.y = event.clientY / window.innerHeight - 0.5;
  };
  window.addEventListener('pointermove', onPointer, { passive: true });
  const onVisibility = () => schedule();
  document.addEventListener('visibilitychange', onVisibility);

  const positions = sparks.geometry.getAttribute('position');
  function render(now: number) {
    frame = 0;
    if (!running || !visible || document.hidden) return;
    const dt = Math.min((now - last) / 1000, 0.05);
    last = now;
    angle += dt * 0.045;
    pointer.smoothX += (pointer.x - pointer.smoothX) * 0.04;
    pointer.smoothY += (pointer.y - pointer.smoothY) * 0.04;
    const radius = 19.5;
    const orbit = angle + pointer.smoothX * 0.35;
    camera.position.set(Math.sin(orbit) * radius, 8.2 - pointer.smoothY * 2.5, Math.cos(orbit) * radius);
    camera.lookAt(target);
    const time = now / 1000;
    flows.forEach((curve, f) => {
      for (let i = 0; i < PER_FLOW; i++) {
        const t = (time * 0.18 + i / PER_FLOW + f * 0.13) % 1;
        const point = curve.getPointAt(t);
        positions.setXYZ(f * PER_FLOW + i, point.x, point.y, point.z);
      }
    });
    positions.needsUpdate = true;
    for (const { ring, phase } of rings) {
      const cycle = (time * 0.55 + phase) % 2.4;
      const grow = cycle < 1.6 ? cycle / 1.6 : 1;
      ring.scale.setScalar(0.4 + grow * 3.4);
      (ring.material as MeshBasicMaterial).opacity = cycle < 1.6 ? (1 - grow) * 0.85 : 0;
    }
    renderer.render(scene, camera);
    if (first) { first = false; onFirstFrame(); }
    schedule();
  }
  function schedule() {
    if (!frame && running && visible && !document.hidden) {
      last = performance.now();
      frame = requestAnimationFrame(render);
    }
  }
  resize();
  schedule();
  return () => {
    running = false;
    cancelAnimationFrame(frame);
    sizeObserver.disconnect();
    viewObserver.disconnect();
    window.removeEventListener('pointermove', onPointer);
    document.removeEventListener('visibilitychange', onVisibility);
    renderer.dispose();
  };
}
