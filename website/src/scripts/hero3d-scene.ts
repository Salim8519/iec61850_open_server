// Procedural low-poly substation for the hero: busbars on gantries, five bays
// (disconnector, breaker, CT, line exit), a transformer and an IED cabinet row.
import {
  BoxGeometry, BufferGeometry, CatmullRomCurve3, CylinderGeometry, EdgesGeometry, Float32BufferAttribute,
  GridHelper, Group, Line, LineBasicMaterial, LineSegments, Mesh, MeshStandardMaterial, Vector3,
} from 'three';

export interface SubstationScene {
  group: Group;
  flows: CatmullRomCurve3[];
  cabinets: Vector3[];
}

const metal = new MeshStandardMaterial({ color: 0x2a3d63, metalness: 0.45, roughness: 0.4 });
const steel = new MeshStandardMaterial({ color: 0x3a4f78, metalness: 0.6, roughness: 0.32 });
const porcelain = new MeshStandardMaterial({ color: 0xc9d4e4, metalness: 0.05, roughness: 0.55 });
const edge = new LineBasicMaterial({ color: 0x6f9be0, transparent: true, opacity: 0.75 });
const conductor = new LineBasicMaterial({ color: 0xa9bddb, transparent: true, opacity: 0.7 });
const led = new MeshStandardMaterial({ color: 0x2be4b0, emissive: 0x2be4b0, emissiveIntensity: 1.6 });

function box(group: Group, w: number, h: number, d: number, x: number, y: number, z: number,
  material = metal, outline = true): Mesh {
  const geometry = new BoxGeometry(w, h, d);
  const mesh = new Mesh(geometry, material);
  mesh.position.set(x, y, z);
  group.add(mesh);
  if (outline) {
    const lines = new LineSegments(new EdgesGeometry(geometry), edge);
    lines.position.copy(mesh.position);
    group.add(lines);
  }
  return mesh;
}

function insulator(group: Group, x: number, y: number, z: number, height = 0.9): void {
  const mesh = new Mesh(new CylinderGeometry(0.09, 0.13, height, 10), porcelain);
  mesh.position.set(x, y + height / 2, z);
  group.add(mesh);
}

function wire(group: Group, points: Vector3[]): void {
  group.add(new Line(new BufferGeometry().setFromPoints(points), conductor));
}

function gantry(group: Group, x: number, z: number, height: number, span: number): void {
  box(group, 0.22, height, 0.22, x, height / 2, z - span / 2, steel);
  box(group, 0.22, height, 0.22, x, height / 2, z + span / 2, steel);
  box(group, 0.22, 0.22, span + 0.22, x, height, z, steel);
}

function bay(group: Group, x: number, flows: CatmullRomCurve3[]): void {
  insulator(group, x - 0.25, 0.6, 0.2);
  insulator(group, x + 0.25, 0.6, 0.2);
  box(group, 0.9, 0.6, 0.5, x, 0.3, 0.2, steel);
  box(group, 0.08, 0.08, 0.9, x, 1.62, 0.2, porcelain, false);
  box(group, 0.9, 1.2, 0.9, x, 0.6, 2.6);
  for (const dz of [-0.28, 0, 0.28]) insulator(group, x, 1.2, 2.6 + dz, 1.0);
  const ct = new Mesh(new CylinderGeometry(0.26, 0.3, 1.0, 14), metal);
  ct.position.set(x, 1.0, 4.6);
  group.add(ct);
  insulator(group, x, 1.5, 4.6, 0.6);
  gantry(group, x, 7, 3.6, 1.6);
  const path = [new Vector3(x, 4.2, -2.3), new Vector3(x, 2.6, -0.8), new Vector3(x, 1.7, 0.2),
    new Vector3(x, 2.3, 2.6), new Vector3(x, 2.2, 4.6), new Vector3(x, 3.6, 7)];
  wire(group, path);
  flows.push(new CatmullRomCurve3(path));
}

function transformer(group: Group, x: number): void {
  box(group, 2.2, 1.9, 1.7, x, 0.95, 2.4);
  for (const dz of [-0.95, 0.95]) box(group, 1.9, 1.5, 0.12, x, 0.85, 2.4 + dz, steel);
  for (const dx of [-0.6, 0, 0.6]) insulator(group, x + dx, 1.9, 2.0, 0.9);
  box(group, 0.9, 0.5, 0.6, x + 0.9, 2.1, 3.1, metal);
}

export function buildSubstation(): SubstationScene {
  const group = new Group();
  const flows: CatmullRomCurve3[] = [];
  const grid = new GridHelper(70, 70, 0x1d3157, 0x101b30);
  (grid.material as LineBasicMaterial).transparent = true;
  (grid.material as LineBasicMaterial).opacity = 0.55;
  group.add(grid);
  for (const z of [-2.3, -3.4]) {
    box(group, 22, 0.12, 0.12, 0, 4.2, z, porcelain, false);
  }
  for (let x = -10.5; x <= 10.5; x += 5.25) gantry(group, x, -2.85, 4.3, 1.9);
  for (const x of [-8, -4, 0, 4]) bay(group, x, flows);
  transformer(group, 8);
  const feed = [new Vector3(8, 4.2, -2.3), new Vector3(8, 3.0, 0.4), new Vector3(8, 2.8, 2.0)];
  wire(group, feed);
  flows.push(new CatmullRomCurve3(feed));
  const cabinets: Vector3[] = [];
  box(group, 13, 0.18, 2.8, 0, 0.09, -9, steel);
  for (let i = 0; i < 8; i++) {
    const x = -5.25 + i * 1.5;
    box(group, 1.1, 2.2, 0.7, x, 1.28, -9);
    const light = new Mesh(new BoxGeometry(0.12, 0.12, 0.04), led);
    light.position.set(x - 0.3, 2.05, -8.63);
    group.add(light);
    cabinets.push(new Vector3(x, 2.5, -9));
  }
  return { group, flows, cabinets };
}

export function pointCloud(count: number): BufferGeometry {
  const geometry = new BufferGeometry();
  geometry.setAttribute('position', new Float32BufferAttribute(new Float32Array(count * 3), 3));
  return geometry;
}
