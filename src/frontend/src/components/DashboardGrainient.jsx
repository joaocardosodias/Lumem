import Grainient from './Grainient'

function DashboardGrainient() {
  return (
    <div className="app-grainient" aria-hidden="true">
      <Grainient
        color1="#efedff"
        color2="#d4c5f8"
        color3="#ccd7ff"
        timeSpeed={1.25}
        colorBalance={0}
        warpStrength={1}
        warpFrequency={5}
        warpSpeed={2}
        warpAmplitude={50}
        blendAngle={0}
        blendSoftness={0.05}
        rotationAmount={500}
        noiseScale={2}
        grainAmount={0.1}
        grainScale={2}
        grainAnimated={false}
        contrast={1.5}
        gamma={1}
        saturation={1}
        centerX={0}
        centerY={0}
        zoom={0.9}
      />
    </div>
  )
}

export default DashboardGrainient
